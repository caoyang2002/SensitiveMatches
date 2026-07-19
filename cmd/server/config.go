package main

import (
	"os"
	"sensitive_matches/pkg/sensitive"

	"time"

	"gopkg.in/yaml.v3"
)

type AIConfigYAML struct {
	AI struct {
		Provider string `yaml:"provider"`
		Config   struct {
			APIKey      string  `yaml:"api-key"`
			BaseURL     string  `yaml:"base-url"`
			Model       string  `yaml:"model"`
			Timeout     int     `yaml:"timeout"` // 毫秒
			MaxRetries  int     `yaml:"max-retries"`
			Temperature float64 `yaml:"temperature"`
			MaxTokens   int     `yaml:"max-tokens"`
		} `yaml:"config"`
	} `yaml:"ai"`
}

func loadAIConfig(configPath string) (*sensitive.AIConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var yamlConfig AIConfigYAML
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, err
	}
	cfg := &sensitive.AIConfig{
		Provider:    yamlConfig.AI.Provider,
		APIKey:      yamlConfig.AI.Config.APIKey,
		BaseURL:     yamlConfig.AI.Config.BaseURL,
		Model:       yamlConfig.AI.Config.Model,
		Timeout:     time.Duration(yamlConfig.AI.Config.Timeout) * time.Millisecond,
		MaxRetries:  yamlConfig.AI.Config.MaxRetries,
		Temperature: yamlConfig.AI.Config.Temperature,
		MaxTokens:   yamlConfig.AI.Config.MaxTokens,
	}
	// 若未配置，使用默认值
	if cfg.Model == "" {
		cfg.Model = "gpt-3.5-turbo"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.1
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 150
	}
	return cfg, nil
}
