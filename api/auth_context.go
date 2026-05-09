package api

import (
	"my_cursor/internal/auth"

	"github.com/gin-gonic/gin"
)

const ctxPrincipalKey = "principal"

// PrincipalFrom 返回当前请求的认证主体（未启用 AUTH 时为本地管理员占位）。
func PrincipalFrom(c *gin.Context) auth.Principal {
	v, ok := c.Get(ctxPrincipalKey)
	if !ok {
		return auth.DevBootstrapPrincipal()
	}
	p, _ := v.(auth.Principal)
	return p
}
