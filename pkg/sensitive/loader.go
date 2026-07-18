package sensitive

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadRules 加载目录下所有 .yml 文件（除特殊文件外）为规则列表
func LoadRules(dir string) ([]Rule, error) {
	var allRules []Rule
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yml") && !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		base := filepath.Base(path)
		if base == "normalize.yml" || base == "whitelist.yml" || base == "blacklist.yml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var container RuleContainer
		if err := yaml.Unmarshal(data, &container); err != nil {
			return fmt.Errorf("文件 %s 解析失败: %w", path, err)
		}
		allRules = append(allRules, container.Rules...)
		return nil
	})
	return allRules, err
}

// LoadWhitelist 加载白名单（仅包含 word 和 action 的简单规则）
func LoadWhitelist(dir string) ([]Rule, error) {
	return loadSimpleRules(dir, "whitelist.yml")
}

// LoadBlacklist 加载黑名单
func LoadBlacklist(dir string) ([]Rule, error) {
	return loadSimpleRules(dir, "blacklist.yml")
}

func loadSimpleRules(dir, filename string) ([]Rule, error) {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var container RuleContainer
	if err := yaml.Unmarshal(data, &container); err != nil {
		return nil, err
	}
	return container.Rules, nil
}

// LoadNormalizeMappings 加载归一化映射（假设格式为 mappings 列表）
func LoadNormalizeMappings(dir string) (map[string]string, error) {
	path := filepath.Join(dir, "normalize.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw struct {
		Mappings []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"mappings"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, pair := range raw.Mappings {
		m[pair.From] = pair.To
	}
	return m, nil
}
