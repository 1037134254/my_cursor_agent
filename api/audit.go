package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Audit 记一条简单审计（操作、资源、用户/租户在 Principal 中另查）。企业可接 ELK/文件落盘。
func Audit(c *gin.Context, action, target string) {
	p := PrincipalFrom(c)
	log.Printf("audit t=%s user=%s action=%s target=%s ip=%s", p.TenantID, p.UserID, action, target, c.ClientIP())
	_ = time.Now() // 预留：可写结构化 JSON
}
