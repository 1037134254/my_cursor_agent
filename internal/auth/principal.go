package auth

import (
	"slices"

	"my_cursor/internal/tenant"
)

const (
	RoleAnon   = "anon"
	RoleViewer = "viewer"
	RoleUser   = "user"
	RoleAdmin  = "admin"
)

// Principal 当前请求身份（JWT 或匿名调试）。
type Principal struct {
	UserID   string
	TenantID string
	Roles    []string
	// AnonymousDebug 为 true 表示未带 JWT 且允许匿名调试（AUTH_ALLOW_ANONYMOUS_DEBUG）。
	AnonymousDebug bool
}

func (p Principal) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}

func (p Principal) HasAny(roles ...string) bool {
	for _, r := range roles {
		if p.HasRole(r) {
			return true
		}
	}
	return false
}

// DevBootstrapPrincipal AUTH 关闭时的合成身份（等价本地管理员，租户 default）。
func DevBootstrapPrincipal() Principal {
	return Principal{
		UserID:   "local",
		TenantID: tenant.DefaultID,
		Roles:    []string{RoleAdmin},
	}
}
