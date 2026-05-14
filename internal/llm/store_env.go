package llm

import (
	"my_cursor/internal/tenant"
	"time"
)

// envProfileStore 当 MySQL 不可用时的只读降级实现：
//
//	直接读现有 LLM_* / ANTHROPIC_* 环境变量，合成一个 default 租户的 chat profile。
//	所有写操作返回 ErrReadOnly，避免管理员误以为改了 env 就会生效。
type envProfileStore struct{}

const envProfileID = "env-default-chat"

func (envProfileStore) Kind() string { return "env" }

// envProfile 由当前 env 合成的只读 profile（每次取最新值，便于 env 热改后重启即用）。
func envProfile() Profile {
	cfg := LoadConfig()
	endpoint := ""
	if len(cfg.ChatEndpoints) > 0 {
		endpoint = cfg.ChatEndpoints[0]
	}
	return Profile{
		ID:                   envProfileID,
		Name:                 "env 默认（自动）",
		Purpose:              PurposeChat,
		Provider:             cfg.Kind,
		Endpoint:             endpoint,
		Model:                cfg.Model,
		APIKeyRef:            "", // env 直读，由 LoadConfig 已塞入 cfg.APIKey；registry 走 envFallback 分支
		TimeoutSeconds:       cfg.TimeoutSeconds,
		MaxRetries:           cfg.MaxRetries,
		RetryBackoffMs:       cfg.RetryBackoffMs,
		RPM:                  cfg.RPM,
		StreamTimeoutSeconds: cfg.StreamTimeoutSecs,
		Enabled:              true,
		CreatedBy:            "system",
		CreatedAt:            time.Unix(0, 0),
		UpdatedAt:            time.Now(),
	}
}

func (envProfileStore) ListProfiles() ([]Profile, error) {
	return []Profile{envProfile()}, nil
}

func (envProfileStore) GetProfile(id string) (Profile, error) {
	if id != envProfileID {
		return Profile{}, ErrProfileNotFound
	}
	return envProfile(), nil
}

func (envProfileStore) UpsertProfile(_ Profile) error { return ErrReadOnly }
func (envProfileStore) DeleteProfile(_ string) error  { return ErrReadOnly }
func (envProfileStore) SetBindings(_ string, _ []TenantBinding) error {
	return ErrReadOnly
}

func (envProfileStore) ListBindings(tenantID string) ([]TenantBinding, error) {
	if tenantID != tenant.DefaultID {
		return nil, nil
	}
	return []TenantBinding{{
		TenantID:  tenant.DefaultID,
		ProfileID: envProfileID,
		Purpose:   PurposeChat,
		IsDefault: true,
		CreatedAt: time.Unix(0, 0),
	}}, nil
}

// DefaultBinding env 模式下：default 租户的 chat 走 env-default-chat，其它一律 ErrBindingNotFound。
func (envProfileStore) DefaultBinding(tenantID string, purpose Purpose) (TenantBinding, Profile, error) {
	if purpose != PurposeChat || tenantID != tenant.DefaultID {
		return TenantBinding{}, Profile{}, ErrBindingNotFound
	}
	return TenantBinding{
		TenantID:  tenant.DefaultID,
		ProfileID: envProfileID,
		Purpose:   PurposeChat,
		IsDefault: true,
		CreatedAt: time.Unix(0, 0),
	}, envProfile(), nil
}
