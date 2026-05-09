package api

import (
	"net/http"
	"strings"

	"my_cursor/internal/auth"
	"my_cursor/internal/tenant"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 解析 Bearer JWT；未启用认证时注入本地开发用管理员身份。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !auth.Enabled() {
			c.Set(ctxPrincipalKey, auth.DevBootstrapPrincipal())
			c.Next()
			return
		}
		h := c.GetHeader("Authorization")
		raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer"))
		if raw != "" {
			p, err := auth.ParseAccessJWT(raw)
			if err == nil {
				c.Set(ctxPrincipalKey, p)
				c.Next()
				return
			}
		}
		if auth.AllowAnonymousDebug() {
			c.Set(ctxPrincipalKey, auth.Principal{
				UserID:         "anonymous",
				TenantID:       tenant.DefaultID,
				Roles:          []string{auth.RoleAnon},
				AnonymousDebug: true,
			})
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或访问令牌无效"})
	}
}

// RequirePermission 拒绝无权限请求（需已执行 AuthMiddleware）。
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Can(PrincipalFrom(c), perm) {
			Audit(c, "deny", perm)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}
		c.Next()
	}
}
