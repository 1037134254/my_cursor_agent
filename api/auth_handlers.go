package api

import (
	"net/http"

	"my_cursor/internal/auth"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

// LoginHandler POST /api/auth/login
func LoginHandler(c *gin.Context) {
	if !auth.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AUTH_ENABLED 未开启"})
		return
	}
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	rec := auth.FindUser(req.Username, req.Password)
	if rec == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	access, err := auth.IssueAccessJWT(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	refresh, err := auth.NewRefreshToken(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	Audit(c, "login", rec.Username)
	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
		"expires_in":    int(auth.AccessTokenTTL().Seconds()),
		"tenant_id":     rec.TenantID,
		"roles":         rec.Roles,
	})
}

// RefreshHandler POST /api/auth/refresh
func RefreshHandler(c *gin.Context) {
	if !auth.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AUTH_ENABLED 未开启"})
		return
	}
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token 必填"})
		return
	}
	rec, ok := auth.ConsumeRefreshToken(req.RefreshToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh 无效或已过期"})
		return
	}
	access, err := auth.IssueAccessJWT(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newRefresh, err := auth.NewRefreshToken(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"access_token":  access,
		"refresh_token": newRefresh,
		"token_type":    "Bearer",
		"expires_in":    int(auth.AccessTokenTTL().Seconds()),
	})
}

// LogoutHandler POST /api/auth/logout
func LogoutHandler(c *gin.Context) {
	var req logoutReq
	_ = c.ShouldBindJSON(&req)
	if req.RefreshToken != "" {
		auth.RevokeRefreshToken(req.RefreshToken)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}
