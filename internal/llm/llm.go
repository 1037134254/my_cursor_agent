package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"my_cursor/internal/tenant"
)

var (
	mu        sync.RWMutex
	defaultGW *Gateway // 旧的"全局唯一" Gateway，仅用于向后兼容工具/测试，不再参与正式推理。
	activeCfg Config
)

// Init 兼容入口（旧代码可能仍调用）；新建议用 InitRegistry。
func Init() {
	_ = SetConfig(LoadConfig())
}

// SetConfig 兼容旧 PUT /api/llm/config：保留全局快照，但生产推理走 registry，不会读这里。
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

// CurrentConfig 返回兼容快照（实际生效以 Profile 为准）。
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

// gatewayFromCtx 从 ctx 中取租户与 purpose（默认 chat），从 registry 找 Gateway；找不到时返回错误。
func gatewayFromCtx(ctx context.Context, purpose Purpose) (*Gateway, error) {
	tid := tenant.FromContext(ctx)
	gw, _, err := Reg().GatewayFor(tid, purpose)
	if err == nil {
		return gw, nil
	}
	// 仅在 env 模式 + default 租户 + 找不到 binding 时，做一次最后兜底（保兼容期）。
	if Reg().store.Kind() == "env" && tid == tenant.DefaultID {
		ensureGW()
		mu.RLock()
		gw := defaultGW
		mu.RUnlock()
		if gw != nil {
			return gw, nil
		}
	}
	return nil, fmt.Errorf("tenant=%s purpose=%s 未找到可用模型 profile：%w", tid, purpose, err)
}

// Chat 非流式：按租户路由 chat profile。
func Chat(ctx context.Context, prompt string) (string, error) {
	gw, err := gatewayFromCtx(ctx, PurposeChat)
	if err != nil {
		return "", err
	}
	return gw.Chat(ctx, prompt)
}

// ChatStream 流式：按租户路由 chat profile。
func ChatStream(ctx context.Context, prompt string, onDelta func(string) error) error {
	gw, err := gatewayFromCtx(ctx, PurposeChat)
	if err != nil {
		return err
	}
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
