package auth

import (
	"os"
	"strings"
	"time"
)

// Enabled 是否启用 JWT 与 RBAC（默认 false，兼容旧部署）。
func Enabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_ENABLED")), "true")
}

// AllowAnonymousDebug 启用认证后，无 JWT 时是否允许匿名调试（租户 default、角色 anon）。
func AllowAnonymousDebug() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_ALLOW_ANONYMOUS_DEBUG")), "true")
}

func jwtSecret() string {
	return strings.TrimSpace(os.Getenv("JWT_SECRET"))
}

// AccessTokenTTL 访问令牌有效期（对外文档/登录响应）。
func AccessTokenTTL() time.Duration {
	return accessTTL()
}

func accessTTL() time.Duration {
	if s := strings.TrimSpace(os.Getenv("JWT_ACCESS_TTL_MINUTES")); s != "" {
		if n, err := time.ParseDuration(s + "m"); err == nil {
			return n
		}
	}
	return 15 * time.Minute
}

func refreshTTL() time.Duration {
	if s := strings.TrimSpace(os.Getenv("JWT_REFRESH_TTL_HOURS")); s != "" {
		if n, err := time.ParseDuration(s + "h"); err == nil {
			return n
		}
	}
	return 168 * time.Hour // 7d
}
