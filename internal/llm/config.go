package llm

import (
	"os"
	"strconv"
	"strings"
)

// ProviderKind 一键切换预设（LLM_PROVIDER）。
type ProviderKind string

const (
	ProviderQwen     ProviderKind = "qwen"
	ProviderOllama   ProviderKind = "ollama"
	ProviderDeepSeek ProviderKind = "deepseek"
	ProviderCustom   ProviderKind = "custom"
)

const (
	defaultOllamaChatURL = "http://127.0.0.1:11434/v1/chat/completions"
	defaultQwenModel     = "qwen2.5-coder:7b-instruct-q4_K_M"
	defaultDeepSeekURL   = "https://api.deepseek.com/v1/chat/completions"
	defaultDeepSeekModel = "deepseek-chat"
)

// Config 网关与下游模型配置（由环境变量加载）。
type Config struct {
	Kind ProviderKind

	// ChatEndpoints 完整 chat/completions URL 列表（单点或多点集群）。
	ChatEndpoints []string
	Model         string
	APIKey        string

	TimeoutSeconds    int
	MaxRetries        int
	RetryBackoffMs    int
	RPM               int
	StreamTimeoutSecs int
}

// LoadConfig 从环境变量读取；Kind 为空时视为 qwen/本地 Ollama。
func LoadConfig() Config {
	kind := ProviderKind(strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER"))))
	if kind == "" {
		kind = ProviderQwen
	}

	cfg := Config{
		Kind:              kind,
		TimeoutSeconds:    envIntOrDefault("LLM_TIMEOUT_SECONDS", 180),
		MaxRetries:        envIntOrDefault("LLM_MAX_RETRIES", 2),
		RetryBackoffMs:    envIntOrDefault("LLM_RETRY_BACKOFF_MS", 400),
		RPM:               0,
		StreamTimeoutSecs: envIntOrDefault("LLM_STREAM_TIMEOUT_SECONDS", 0),
	}
	if s := strings.TrimSpace(os.Getenv("LLM_RPM")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			cfg.RPM = n
		}
	}

	cluster := strings.TrimSpace(os.Getenv("LLM_CLUSTER_ENDPOINTS"))
	if cluster != "" {
		for _, p := range strings.Split(cluster, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.ChatEndpoints = append(cfg.ChatEndpoints, p)
			}
		}
	}

	switch kind {
	case ProviderQwen, ProviderOllama:
		if len(cfg.ChatEndpoints) == 0 {
			cfg.ChatEndpoints = []string{envOrDefault("LLM_API_URL", defaultOllamaChatURL)}
		}
		cfg.Model = envOrDefault("LLM_MODEL", defaultQwenModel)
		cfg.APIKey = strings.TrimSpace(os.Getenv("LLM_API_KEY"))

	case ProviderDeepSeek:
		if len(cfg.ChatEndpoints) == 0 {
			u := envOrDefault("DEEPSEEK_API_URL", "")
			if u == "" {
				u = envOrDefault("LLM_API_URL", defaultDeepSeekURL)
			}
			cfg.ChatEndpoints = []string{u}
		}
		cfg.Model = envOrDefault("DEEPSEEK_MODEL", envOrDefault("LLM_MODEL", defaultDeepSeekModel))
		cfg.APIKey = strings.TrimSpace(envOrDefault("DEEPSEEK_API_KEY", envOrDefault("LLM_API_KEY", "")))

	case ProviderCustom:
		if len(cfg.ChatEndpoints) == 0 {
			cfg.ChatEndpoints = []string{envOrDefault("LLM_API_URL", defaultOllamaChatURL)}
		}
		cfg.Model = envOrDefault("LLM_MODEL", defaultQwenModel)
		cfg.APIKey = strings.TrimSpace(os.Getenv("LLM_API_KEY"))

	default:
		if len(cfg.ChatEndpoints) == 0 {
			cfg.ChatEndpoints = []string{envOrDefault("LLM_API_URL", defaultOllamaChatURL)}
		}
		cfg.Model = envOrDefault("LLM_MODEL", defaultQwenModel)
		cfg.APIKey = strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	}

	return cfg
}

func envOrDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func envIntOrDefault(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}
