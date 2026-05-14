package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"my_cursor/internal/llm"

	"github.com/gin-gonic/gin"
)

type profileView struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Purpose              string `json:"purpose"`
	Provider             string `json:"provider"`
	Endpoint             string `json:"endpoint"`
	Model                string `json:"model"`
	APIKeyRef            string `json:"api_key_ref"`
	APIKeyResolved       bool   `json:"api_key_resolved"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	MaxRetries           int    `json:"max_retries"`
	RetryBackoffMs       int    `json:"retry_backoff_ms"`
	RPM                  int    `json:"rpm"`
	StreamTimeoutSeconds int    `json:"stream_timeout_seconds"`
	Enabled              bool   `json:"enabled"`
	CreatedBy            string `json:"created_by"`
	CreatedAt            int64  `json:"created_at"`
	UpdatedAt            int64  `json:"updated_at"`
}

type profileUpsertReq struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Purpose              string `json:"purpose"`
	Provider             string `json:"provider"`
	Endpoint             string `json:"endpoint"`
	Model                string `json:"model"`
	APIKeyRef            string `json:"api_key_ref"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	MaxRetries           int    `json:"max_retries"`
	RetryBackoffMs       int    `json:"retry_backoff_ms"`
	RPM                  int    `json:"rpm"`
	StreamTimeoutSeconds int    `json:"stream_timeout_seconds"`
	Enabled              *bool  `json:"enabled"`
}

func toProfileView(p llm.Profile) profileView {
	return profileView{
		ID:                   p.ID,
		Name:                 p.Name,
		Purpose:              string(p.Purpose),
		Provider:             string(p.Provider),
		Endpoint:             p.Endpoint,
		Model:                p.Model,
		APIKeyRef:            p.APIKeyRef,
		APIKeyResolved:       p.APIKeyRef != "" && strings.TrimSpace(envGet(p.APIKeyRef)) != "",
		TimeoutSeconds:       p.TimeoutSeconds,
		MaxRetries:           p.MaxRetries,
		RetryBackoffMs:       p.RetryBackoffMs,
		RPM:                  p.RPM,
		StreamTimeoutSeconds: p.StreamTimeoutSeconds,
		Enabled:              p.Enabled,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            p.CreatedAt.Unix(),
		UpdatedAt:            p.UpdatedAt.Unix(),
	}
}

// envGet 透传 os.Getenv，便于单测注入。
var envGet = func(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// LLMProfilesListHandler GET /api/llm/profiles
func LLMProfilesListHandler(c *gin.Context) {
	profs, err := llm.Reg().Store().ListProfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]profileView, 0, len(profs))
	for _, p := range profs {
		out = append(out, toProfileView(p))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "profiles": out, "store": llm.Reg().Store().Kind()})
}

// LLMProfileUpsertHandler POST /api/llm/profiles 或 PUT /api/llm/profiles/:id
func LLMProfileUpsertHandler(c *gin.Context) {
	var req profileUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if pid := strings.TrimSpace(c.Param("id")); pid != "" {
		req.ID = pid
	}

	p := llm.Profile{
		ID:                   req.ID,
		Name:                 req.Name,
		Purpose:              llm.Purpose(req.Purpose),
		Provider:             llm.ProviderKind(req.Provider),
		Endpoint:             req.Endpoint,
		Model:                req.Model,
		APIKeyRef:            req.APIKeyRef,
		TimeoutSeconds:       req.TimeoutSeconds,
		MaxRetries:           req.MaxRetries,
		RetryBackoffMs:       req.RetryBackoffMs,
		RPM:                  req.RPM,
		StreamTimeoutSeconds: req.StreamTimeoutSeconds,
		Enabled:              true,
		CreatedBy:            PrincipalFrom(c).UserID,
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if err := p.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := llm.Reg().Store().UpsertProfile(p); err != nil {
		if errors.Is(err, llm.ErrReadOnly) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry 处于只读模式（未连接 MySQL），无法写入"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	llm.Reg().EvictProfile(p.ID)
	Audit(c, "llm_profile_upsert", p.ID)

	got, err := llm.Reg().Store().GetProfile(p.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "profile": toProfileView(got)})
}

// LLMProfileDeleteHandler DELETE /api/llm/profiles/:id
func LLMProfileDeleteHandler(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 id"})
		return
	}
	if err := llm.Reg().Store().DeleteProfile(id); err != nil {
		switch {
		case errors.Is(err, llm.ErrReadOnly):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry 只读"})
		case errors.Is(err, llm.ErrProfileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, llm.ErrProfileInUse):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	llm.Reg().EvictProfile(id)
	Audit(c, "llm_profile_delete", id)
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// LLMProfileProbeHandler POST /api/llm/profiles/:id/probe
// 用 Gateway 跑一次极简对话，校验配置 & API Key 是否真的能通；结果不算入业务日志。
func LLMProfileProbeHandler(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 id"})
		return
	}
	prof, err := llm.Reg().Store().GetProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	cfg := prof.EffectiveConfig(0)
	cfg.APIKey = strings.TrimSpace(envGet(prof.APIKeyRef))
	if cfg.APIKey == "" && prof.APIKeyRef != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "api_key_ref 指向的环境变量未设置或为空: " + prof.APIKeyRef,
		})
		return
	}
	if cfg.TimeoutSeconds > 30 {
		cfg.TimeoutSeconds = 30
	}
	gw := llm.NewGateway(cfg)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	reply, err := gw.Chat(ctx, "ping")
	if err != nil {
		Audit(c, "llm_profile_probe_fail", id)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	Audit(c, "llm_profile_probe_ok", id)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"profile": toProfileView(prof),
		"reply":   reply,
	})
}
