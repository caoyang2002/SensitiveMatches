package sensitive

import (
	"log"
	"strings"
	"sync"
)

type Checker struct {
	normalizer *Normalizer
	matcher    *Matcher
	whitelist  []Rule // 白名单规则（用于快速判断是否命中）
	blacklist  []Rule // 黑名单规则
	mu         sync.RWMutex
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

	// 白名单和黑名单也加入匹配器？为了统一匹配，但我们希望它们单独处理
	// 但为了匹配，我们也可以把它们作为普通规则加入，但设置特殊标志。为避免冲突，我们将白/黑名单作为独立列表。
	// 但匹配时需要查找文本中是否包含这些词，我们可以用同样的 Matcher 但只针对 word 字段。
	// 做法：构造一个仅包含白/黑名单规则的 Matcher，用于快速判断。
	// allSpecial := append(whitelist, blacklist...)
	// specialMatcher := NewMatcher(allSpecial)

	// 普通规则 matcher
	normalMatcher := NewMatcher(rules)
	log.Printf("加载完成：普通规则 %d 条，白名单 %d 条，黑名单 %d 条", len(rules), len(whitelist), len(blacklist))
	return &Checker{
		normalizer: normalizer,
		matcher:    normalMatcher,
		whitelist:  whitelist,
		blacklist:  blacklist,
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
	whiteHits := c.matchSpecial(c.whitelist, normalized)
	blackHits := c.matchSpecial(c.blacklist, normalized)

	// 合并所有匹配
	allMatches := append(matches, whiteHits...)
	allMatches = append(allMatches, blackHits...)

	// 4. 决策
	finalAction := ActionPass
	totalScore := 0
	hasBlack := false
	hasWhite := false
	for _, m := range allMatches {
		// 累计分数（只累计普通规则和白/黑的分数，白名单分数可能为0）
		if m.Action != ActionPass { // 白名单通常 action=pass，不计分或计分低，但这里我们累加所有分数
			totalScore += m.Score
		}
		if m.Action == ActionBlock {
			hasBlack = true
		}
		if m.Action == ActionPass && m.Score == 0 { // 白名单一般 action=pass, score=0
			hasWhite = true
		}
	}

	// 黑名单强制 block
	if hasBlack {
		finalAction = ActionBlock
	} else if hasWhite {
		finalAction = ActionPass // 白名单覆盖其他
	} else {
		// 根据普通规则动作决定
		for _, m := range allMatches {
			if m.Action == ActionBlock {
				finalAction = ActionBlock
				break
			}
			if m.Action == ActionReview && finalAction != ActionBlock {
				finalAction = ActionReview
			}
		}
	}

	// 计算等级
	level := calcLevel(totalScore)

	// 敏感词替换
	masked := text
	if finalAction != ActionPass {
		// 按命中词替换（仅替换普通规则，白/黑不替换）
		for _, m := range matches {
			if m.RuleID != "" { // 普通规则有ID
				// 替换为 * 号
				stars := strings.Repeat("*", len([]rune(m.Word)))
				masked = strings.Replace(masked, m.Word, stars, 1) // 简单替换，处理多个相同词需要全局替换
			}
		}
		// 实际上需要正确处理重叠和多次出现，这里简化
		// 更好的做法：按位置替换，避免重复替换
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
func (c *Checker) matchSpecial(rules []Rule, text string) []*MatchResult {
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
			res = append(res, &MatchResult{
				RuleID:   rule.ID,
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
