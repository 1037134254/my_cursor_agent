package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

var (
	mu        sync.RWMutex
	defaultGW *Gateway
	activeCfg Config
)

// Init 读取环境变量并初始化默认网关（应在 main 启动时调用一次）。
func Init() {
	_ = SetConfig(LoadConfig())
}

// SetConfig 允许在运行时切换模型配置（页面可调用 API 热更新，无需重启）。
func SetConfig(cfg Config) error {
	normalized, err := normalizeConfig(cfg)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	activeCfg = normalized
	defaultGW = NewGateway(normalized)
	return nil
}

// CurrentConfig 返回当前生效配置副本。
func CurrentConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return activeCfg
}

func ensureGW() {
	mu.RLock()
	ok := defaultGW != nil
	mu.RUnlock()
	if ok {
		return
	}
	_ = SetConfig(LoadConfig())
}

// Chat 非流式对话（默认 Provider + 超时/重试/限流）。
func Chat(prompt string) (string, error) {
	ensureGW()
	mu.RLock()
	gw := defaultGW
	mu.RUnlock()
	return gw.Chat(context.Background(), prompt)
}

// ChatStream 流式对话（ctx 可由上层控制总时长）。
func ChatStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	ensureGW()
	mu.RLock()
	gw := defaultGW
	mu.RUnlock()
	return gw.ChatStream(ctx, prompt, onDelta)
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.Kind = ProviderKind(strings.ToLower(strings.TrimSpace(string(cfg.Kind))))
	switch cfg.Kind {
	case "":
		cfg.Kind = ProviderCustom
	case ProviderQwen, ProviderOllama, ProviderDeepSeek, ProviderCustom:
	default:
		return Config{}, fmt.Errorf("unsupported provider: %s", cfg.Kind)
	}

	if len(cfg.ChatEndpoints) == 0 || strings.TrimSpace(cfg.ChatEndpoints[0]) == "" {
		switch cfg.Kind {
		case ProviderDeepSeek:
			cfg.ChatEndpoints = []string{defaultDeepSeekURL}
		case ProviderCustom:
			cfg.ChatEndpoints = []string{defaultGLMURL}
		default:
			cfg.ChatEndpoints = []string{defaultOllamaChatURL}
		}
	}
	for i := range cfg.ChatEndpoints {
		cfg.ChatEndpoints[i] = strings.TrimSpace(cfg.ChatEndpoints[i])
	}
	if strings.TrimSpace(cfg.Model) == "" {
		if cfg.Kind == ProviderDeepSeek {
			cfg.Model = defaultDeepSeekModel
		} else if cfg.Kind == ProviderCustom {
			cfg.Model = defaultGLMModel
		} else {
			cfg.Model = defaultQwenModel
		}
	}
	cfg.APIKey = cleanSecret(cfg.APIKey)
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 180
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryBackoffMs < 0 {
		cfg.RetryBackoffMs = 0
	}
	if cfg.RPM < 0 {
		cfg.RPM = 0
	}
	if cfg.StreamTimeoutSecs < 0 {
		cfg.StreamTimeoutSecs = 0
	}
	return cfg, nil
}
