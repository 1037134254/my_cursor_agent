package llm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Purpose 模型用途。
type Purpose string

const (
	PurposeChat   Purpose = "chat"
	PurposeEmbed  Purpose = "embed"
	PurposeRerank Purpose = "rerank"
)

// ValidPurpose 是否为受支持的用途。
func ValidPurpose(p Purpose) bool {
	switch p {
	case PurposeChat, PurposeEmbed, PurposeRerank:
		return true
	}
	return false
}

var profileIDRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,62}$`)

// Profile 模型档案（数据库 model_profile 表的对象表示）。
// 注意：APIKeyRef 仅存 env 变量名（如 "GLM_KEY_A"），真值由 os.Getenv 读取，
// 数据库永远不出现明文密钥；详见 .env.example 中的 api_key_ref 用法说明。
type Profile struct {
	ID                   string
	Name                 string
	Purpose              Purpose
	Provider             ProviderKind
	Endpoint             string
	Model                string
	APIKeyRef            string
	TimeoutSeconds       int
	MaxRetries           int
	RetryBackoffMs       int
	RPM                  int
	StreamTimeoutSeconds int
	Enabled              bool
	CreatedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Validate 写库前做静态校验并补默认值（就地修改）。
func (p *Profile) Validate() error {
	p.ID = strings.TrimSpace(p.ID)
	if !profileIDRegex.MatchString(p.ID) {
		return errors.New("profile id 必须为字母开头、长度 2~63，字符仅含字母数字下划线连字符")
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("profile name 不能为空")
	}
	if !ValidPurpose(p.Purpose) {
		return fmt.Errorf("不支持的 purpose: %s", p.Purpose)
	}
	p.Provider = ProviderKind(strings.ToLower(strings.TrimSpace(string(p.Provider))))
	switch p.Provider {
	case ProviderQwen, ProviderOllama, ProviderDeepSeek, ProviderCustom:
	case "":
		p.Provider = ProviderCustom
	default:
		return fmt.Errorf("不支持的 provider: %s", p.Provider)
	}
	p.Endpoint = strings.TrimSpace(p.Endpoint)
	if !strings.HasPrefix(p.Endpoint, "http://") && !strings.HasPrefix(p.Endpoint, "https://") {
		return errors.New("endpoint 必须以 http:// 或 https:// 开头")
	}
	p.Model = strings.TrimSpace(p.Model)
	if p.Model == "" {
		return errors.New("model 不能为空")
	}
	p.APIKeyRef = strings.TrimSpace(p.APIKeyRef)
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 180
	}
	if p.MaxRetries < 0 {
		p.MaxRetries = 0
	}
	if p.RetryBackoffMs < 0 {
		p.RetryBackoffMs = 0
	}
	if p.RPM < 0 {
		p.RPM = 0
	}
	if p.StreamTimeoutSeconds < 0 {
		p.StreamTimeoutSeconds = 0
	}
	return nil
}

// TenantBinding 租户与 profile 的绑定关系（含按租户覆盖的 RPM）。
type TenantBinding struct {
	TenantID    string
	ProfileID   string
	Purpose     Purpose
	IsDefault   bool
	RPMOverride int
	CreatedAt   time.Time
}

// EffectiveConfig 根据 binding 中的 RPM override 计算真正生效的 Config。
func (p Profile) EffectiveConfig(rpmOverride int) Config {
	cfg := Config{
		Kind:              p.Provider,
		ChatEndpoints:     []string{p.Endpoint},
		Model:             p.Model,
		TimeoutSeconds:    p.TimeoutSeconds,
		MaxRetries:        p.MaxRetries,
		RetryBackoffMs:    p.RetryBackoffMs,
		RPM:               p.RPM,
		StreamTimeoutSecs: p.StreamTimeoutSeconds,
	}
	if rpmOverride > 0 {
		cfg.RPM = rpmOverride
	}
	return cfg
}

// ProfileStore 模型注册表持久化接口。
type ProfileStore interface {
	Kind() string

	ListProfiles() ([]Profile, error)
	GetProfile(id string) (Profile, error)
	UpsertProfile(p Profile) error
	DeleteProfile(id string) error

	ListBindings(tenantID string) ([]TenantBinding, error)
	SetBindings(tenantID string, bs []TenantBinding) error
	DefaultBinding(tenantID string, purpose Purpose) (TenantBinding, Profile, error)
}

var (
	ErrProfileNotFound = errors.New("profile 不存在")
	ErrBindingNotFound = errors.New("当前租户未绑定该用途的默认 profile")
	ErrReadOnly        = errors.New("当前 registry 为只读（未连接 MySQL）")
	ErrProfileInUse    = errors.New("profile 正在被租户绑定，无法删除")
)
