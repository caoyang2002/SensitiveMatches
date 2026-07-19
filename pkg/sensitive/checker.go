package sensitive

import (
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type PolicyOverride struct {
	Action Action
	Score  int
}

type Checker struct {
	normalizer     *Normalizer
	matcher        *Matcher
	whitelist      []Rule // 白名单规则（用于快速判断是否命中）
	blacklist      []Rule // 黑名单规则
	categoryPolicy map[string]PolicyOverride
	mu             sync.RWMutex
}

// NewChecker 创建审核器
func NewChecker(dictDir string) (*Checker, error) {
	// 加载归一化映射
	mappings, err := LoadNormalizeMappings(dictDir)
	if err != nil {
		return nil, err
	}
	normalizer := NewNormalizer(mappings)

	// 加载普通规则
	rules, err := LoadRules(dictDir)
	if err != nil {
		return nil, err
	}
	// 加载白名单和黑名单（作为特殊规则）
	whitelist, err := LoadWhitelist(dictDir)
	if err != nil {
		return nil, err
	}
	blacklist, err := LoadBlacklist(dictDir)
	if err != nil {
		return nil, err
	}

	// 普通规则 matcher
	normalMatcher := NewMatcher(rules)

	// 加载策略
	policyPath := filepath.Join(dictDir, "policy.yml")
	categoryMap, err := LoadPolicy(policyPath)
	if err != nil {
		return nil, err
	}
	// 转换为 PolicyOverride
	policyOverride := make(map[string]PolicyOverride)
	for cat, info := range categoryMap {
		policyOverride[cat] = PolicyOverride{
			Action: Action(info.Action),
			Score:  info.Score,
		}
	}

	// 记录加载信息
	log.Printf("加载完成：普通规则 %d 条，白名单 %d 条，黑名单 %d 条，策略 %d 条",
		len(rules), len(whitelist), len(blacklist), len(policyOverride))

	return &Checker{
		normalizer:     normalizer,
		matcher:        normalMatcher,
		whitelist:      whitelist,
		blacklist:      blacklist,
		categoryPolicy: policyOverride,
	}, nil
}

// Reload 热更新
func (c *Checker) Reload(dictDir string) error {
	newChecker, err := NewChecker(dictDir)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.normalizer = newChecker.normalizer
	c.matcher = newChecker.matcher
	c.whitelist = newChecker.whitelist
	c.blacklist = newChecker.blacklist
	return nil
}

// Check 执行审核
func (c *Checker) Check(text string) *CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 1. 归一化
	normalized := c.normalizer.Normalize(text)

	// 2. 匹配普通规则
	matches := c.matcher.Match(normalized)

	// 3. 白/黑名单匹配（简单词匹配，因为白/黑名单只有 word 字段）
	// 我们可以复用 c.matcher 但它的 wordMap 包括了特殊规则，但可能混入普通规则？
	// 更安全：单独检查白/黑名单
	whiteHits := c.matchSpecial(c.whitelist, normalized, "WHITE")
	blackHits := c.matchSpecial(c.blacklist, normalized, "BLACK")

	// 合并所有匹配
	allMatches := append(matches, whiteHits...)
	allMatches = append(allMatches, blackHits...)

	// 应用策略覆盖（仅对普通规则，白/黑名单不覆盖）
	for _, m := range allMatches {
		// 跳过白/黑名单（通过 RuleID 前缀或来源标记）
		if strings.HasPrefix(m.RuleID, "WHITE_") || strings.HasPrefix(m.RuleID, "BLACK_") {
			continue
		}
		if override, ok := c.categoryPolicy[m.Category]; ok {
			m.Action = override.Action
			m.Score = override.Score
		}
	}

	// 4. 决策
	finalAction := ActionPass
	totalScore := 0
	hasBlack := false
	hasWhite := false

	// 先累积所有分数
	for _, m := range allMatches {
		totalScore += m.Score
		if strings.HasPrefix(m.RuleID, "BLACK_") {
			hasBlack = true
		}
		if strings.HasPrefix(m.RuleID, "WHITE_") {
			hasWhite = true
		}
	}

	// 优先级：黑名单 > 白名单 > 其他
	if hasBlack {
		finalAction = ActionBlock
	} else if hasWhite {
		finalAction = ActionPass
	} else {
		// 取最高风险动作
		highest := ActionPass
		for _, m := range allMatches {
			switch m.Action {
			case ActionBlock:
				highest = ActionBlock
			case ActionReview:
				if highest != ActionBlock {
					highest = ActionReview
				}
			case ActionReplace:
				if highest != ActionBlock && highest != ActionReview {
					highest = ActionReplace
				}
			case ActionShadow:
				if highest == ActionPass {
					highest = ActionShadow
				}
			}
		}
		finalAction = highest
	}

	// 计算等级
	level := calcLevel(totalScore)

	// 敏感词替换（基于归一化文本，避免字节索引截断）
	masked := normalized
	if finalAction != ActionPass && len(matches) > 0 {
		type replaceItem struct{ start, end int } // 只记录字节位置
		var replaces []replaceItem

		for _, m := range matches {
			if m.RuleID != "" && !strings.HasPrefix(m.RuleID, "WHITE_") && !strings.HasPrefix(m.RuleID, "BLACK_") {
				if m.Start >= 0 && m.End <= len(normalized) && m.Start < m.End {
					replaces = append(replaces, replaceItem{m.Start, m.End})
				}
			}
		}

		// 按 start 降序排序，从后往前替换
		sort.Slice(replaces, func(i, j int) bool { return replaces[i].start > replaces[j].start })

		// 将归一化文本转为 rune 切片，便于安全替换
		runes := []rune(normalized)
		for _, r := range replaces {
			// 将字节索引转换为字符索引
			charStart := utf8.RuneCountInString(normalized[:r.start])
			charEnd := utf8.RuneCountInString(normalized[:r.end])
			if charStart >= 0 && charEnd <= len(runes) && charStart < charEnd {
				stars := strings.Repeat("*", charEnd-charStart)
				// 替换 rune 切片
				newRunes := append(runes[:charStart], append([]rune(stars), runes[charEnd:]...)...)
				runes = newRunes
			}
		}
		masked = string(runes)
	}

	return &CheckResult{
		Original:  text,
		Masked:    masked,
		Sensitive: finalAction != ActionPass,
		Level:     level,
		Score:     totalScore,
		Action:    finalAction,
		Matches:   allMatches,
	}
}

// matchSpecial 对给定规则列表进行精确词匹配
// func (c *Checker) matchSpecial(rules []Rule, text string) []*MatchResult {
// 	var res []*MatchResult
// 	for _, rule := range rules {
// 		if rule.Word == "" {
// 			continue
// 		}
// 		pos := 0
// 		for {
// 			idx := strings.Index(text[pos:], rule.Word)
// 			if idx == -1 {
// 				break
// 			}
// 			start := pos + idx
// 			end := start + len(rule.Word)
// 			res = append(res, &MatchResult{
// 				RuleID:   rule.ID,
// 				Category: rule.Category,
// 				Word:     rule.Word,
// 				Start:    start,
// 				End:      end,
// 				Action:   rule.Action,
// 				Score:    rule.Score,
// 				Tags:     rule.Tags,
// 			})
// 			pos = start + len(rule.Word)
// 		}
// 	}
// 	return res
// }

func calcLevel(score int) Level {
	switch {
	case score <= 20:
		return LevelNormal
	case score <= 50:
		return LevelLow
	case score <= 80:
		return LevelMedium
	case score <= 100:
		return LevelHigh
	default:
		return LevelCritical
	}
}

func (c *Checker) matchSpecial(rules []Rule, text string, prefix string) []*MatchResult {
	var res []*MatchResult
	for _, rule := range rules {
		if rule.Word == "" {
			continue
		}
		pos := 0
		for {
			idx := strings.Index(text[pos:], rule.Word)
			if idx == -1 {
				break
			}
			start := pos + idx
			end := start + len(rule.Word)
			ruleID := rule.ID
			if ruleID == "" {
				ruleID = prefix + "_" + rule.Category
			} else {
				ruleID = prefix + "_" + ruleID
			}
			res = append(res, &MatchResult{
				RuleID:   ruleID,
				Category: rule.Category,
				Word:     rule.Word,
				Start:    start,
				End:      end,
				Action:   rule.Action,
				Score:    rule.Score,
				Tags:     rule.Tags,
			})
			pos = start + len(rule.Word)
		}
	}
	return res
}
