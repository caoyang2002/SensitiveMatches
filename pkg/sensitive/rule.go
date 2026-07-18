package sensitive

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// StringSlice 支持字符串或字符串列表的 YAML 解析
type StringSlice []string

func (s *StringSlice) UnmarshalYAML(value *yaml.Node) error {
	// 尝试解码为字符串
	var str string
	if err := value.Decode(&str); err == nil {
		*s = []string{str}
		return nil
	}
	// 尝试解码为字符串切片
	var slice []string
	if err := value.Decode(&slice); err == nil {
		*s = slice
		return nil
	}
	return fmt.Errorf("期望字符串或字符串列表")
}

type Action string

const (
	ActionPass   Action = "pass"
	ActionReview Action = "review"
	ActionBlock  Action = "block"
)

type Level string

const (
	LevelNormal   Level = "Normal"
	LevelLow      Level = "Low"
	LevelMedium   Level = "Medium"
	LevelHigh     Level = "High"
	LevelCritical Level = "Critical"
)

// Rule 单条规则
type Rule struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Category    string      `yaml:"category"`
	Priority    int         `yaml:"priority"`
	Action      Action      `yaml:"action"`
	Score       int         `yaml:"score"`
	Regex       StringSlice `yaml:"regex"` // 支持字符串或数组
	Tags        StringSlice `yaml:"tags"`  // 支持字符串或数组
	Replace     bool        `yaml:"replace"`
	Description string      `yaml:"description"`
	Examples    StringSlice `yaml:"examples"`       // 支持字符串或数组
	Word        string      `yaml:"word,omitempty"` // 白/黑名单专用
}

// RuleContainer ...
type RuleContainer struct {
	Version string `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

// MatchResult ...
type MatchResult struct {
	RuleID   string   `json:"rule_id"`
	Category string   `json:"category"`
	Word     string   `json:"word"`
	Start    int      `json:"start"`
	End      int      `json:"end"`
	Action   Action   `json:"action"`
	Score    int      `json:"score"`
	Tags     []string `json:"tags"` // 使用时将 StringSlice 转为 []string
}

// CheckResult ...
type CheckResult struct {
	Original  string         `json:"original"`
	Masked    string         `json:"masked"`
	Sensitive bool           `json:"sensitive"`
	Level     Level          `json:"level"`
	Score     int            `json:"score"`
	Action    Action         `json:"action"`
	Matches   []*MatchResult `json:"matches"`
}
