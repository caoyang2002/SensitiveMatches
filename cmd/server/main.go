package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"sensitive_matches/pkg/sensitive"
)

var checker *sensitive.Checker

func main() {
	dictDir := "sensitive-dicts"

	if _, err := os.Stat(dictDir); os.IsNotExist(err) {
		dictDir = os.Getenv("DICT_DIR")
		if dictDir == "" {
			log.Fatal("词典目录不存在，请设置 DICT_DIR 环境变量")
		}
	}

	// 加载 LLM 配置（若 config.yml 存在）
	var llmConfig *sensitive.AIConfig
	configPath := "config.yml"
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := loadAIConfig(configPath)
		if err != nil {
			log.Printf("加载 config.yml 失败: %v，LLM 复判将禁用", err)
		} else if cfg.APIKey != "" {
			llmConfig = cfg
			log.Printf("LLM 复判已启用，提供商: %s, 模型: %s", cfg.Provider, cfg.Model)
		} else {
			log.Println("config.yml 中未提供 API Key，LLM 复判禁用")
		}
	} else {
		log.Println("未找到 config.yml，LLM 复判禁用")
	}

	var err error
	checker, err = sensitive.NewChecker(dictDir, llmConfig)
	if err != nil {
		log.Fatalf("初始化审核器失败: %v", err)
	}
	log.Printf("词典加载成功，目录: %s", dictDir)

	http.HandleFunc("/check", checkHandler)
	http.HandleFunc("/reload", reloadHandler)
	log.Println("服务启动在 :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type CheckRequest struct {
	Text string `json:"text"`
}

type CheckResponse struct {
	Original  string                   `json:"original"`
	Masked    string                   `json:"masked"`
	Sensitive bool                     `json:"sensitive"`
	Level     sensitive.Level          `json:"level"`
	Score     int                      `json:"score"`
	Action    sensitive.Action         `json:"action"`
	Matches   []*sensitive.MatchResult `json:"matches"`
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效 JSON", http.StatusBadRequest)
		return
	}
	result := checker.Check(req.Text)
	resp := CheckResponse{
		Original:  result.Original,
		Masked:    result.Masked,
		Sensitive: result.Sensitive,
		Level:     result.Level,
		Score:     result.Score,
		Action:    result.Action,
		Matches:   result.Matches,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}
	dictDir := "sensitive-dicts"
	llmConfig := &sensitive.AIConfig{
		Enable:      os.Getenv("ENABLELLM") == "true",
		Provider:    os.Getenv("PROVIDER"),
		APIKey:      os.Getenv("API_KEY"),
		BaseURL:     os.Getenv("BASE_URL"),
		Model:       getEnvOrDefault("MODEL", "gpt-3.5-turbo"),
		Timeout:     30,
		MaxRetries:  2,
		Temperature: 0.1,
		MaxTokens:   150,
	}

	if err := checker.Reload(dictDir, llmConfig); err != nil {
		http.Error(w, "重载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("重载成功"))
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
