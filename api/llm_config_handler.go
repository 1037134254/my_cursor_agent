package api

import (
	"net/http"
	"strings"

	"my_cursor/internal/llm"
	"my_cursor/internal/tenant"

	"github.com/gin-gonic/gin"
)

type llmConfigPatchReq struct {
	Provider       string  `json:"provider"`
	APIURL         string  `json:"api_url"`
	Model          string  `json:"model"`
	APIKey         *string `json:"api_key"`
	TimeoutSeconds *int    `json:"timeout_seconds"`
	MaxRetries     *int    `json:"max_retries"`
	RetryBackoffMs *int    `json:"retry_backoff_ms"`
	RPM            *int    `json:"rpm"`
}

type llmConfigView struct {
	Provider       string `json:"provider"`
	APIURL         string `json:"api_url"`
	Model          string `json:"model"`
	HasAPIKey      bool   `json:"has_api_key"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	MaxRetries     int    `json:"max_retries"`
	RetryBackoffMs int    `json:"retry_backoff_ms"`
	RPM            int    `json:"rpm"`
}

// LLMConfigGetHandler 兼容旧接口：返回 default 租户的 default chat profile 的"配置视图"。
// 真正的多租户配置请改用 GET /api/llm/current 或 GET /api/llm/profiles。
func LLMConfigGetHandler(c *gin.Context) {
	store := llm.Reg().Store()
	// env 模式：维持旧行为，从 LoadConfig 出一个视图。
	if store.Kind() == "env" {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"config":     toConfigView(llm.CurrentConfig()),
			"deprecated": "GET /api/llm/config 已废弃，请改用 GET /api/llm/current",
		})
		return
	}
	_, prof, err := store.DefaultBinding(tenant.DefaultID, llm.PurposeChat)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"config":     llmConfigView{},
			"deprecated": "GET /api/llm/config 已废弃；当前 default 租户未绑定 chat profile，请用 POST /api/llm/profiles 创建后绑定",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"config":     profileAsConfigView(prof),
		"deprecated": "GET /api/llm/config 已废弃，请改用 GET /api/llm/current 或 GET /api/llm/profiles",
	})
}

// LLMConfigPutHandler 兼容旧接口：
//   - env 模式：保留原"修改进程级 cfg"行为（仅开发期有效，env 模式无持久化）。
//   - mysql 模式：拒绝并提示走新 API（避免静默写入 default 租户 profile 引发误解）。
func LLMConfigPutHandler(c *gin.Context) {
	store := llm.Reg().Store()
	if store.Kind() != "env" {
		c.JSON(http.StatusGone, gin.H{
			"error": "PUT /api/llm/config 已废弃；模型管理请改用 POST/PUT /api/llm/profiles + PUT /api/tenants/:tid/models",
		})
		return
	}

	var req llmConfigPatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.APIKey != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "安全模式已启用：请使用服务端环境变量配置 API Key"})
		return
	}

	cfg := llm.CurrentConfig()
	if p := strings.TrimSpace(req.Provider); p != "" {
		cfg.Kind = llm.ProviderKind(strings.ToLower(p))
	}
	if u := strings.TrimSpace(req.APIURL); u != "" {
		if !strings.Contains(u, "/chat/completions") {
			u = llm.AnthropicBaseToChatURL(u)
		}
		cfg.ChatEndpoints = []string{u}
	}
	if m := strings.TrimSpace(req.Model); m != "" {
		cfg.Model = m
	}
	if req.TimeoutSeconds != nil {
		cfg.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.MaxRetries != nil {
		cfg.MaxRetries = *req.MaxRetries
	}
	if req.RetryBackoffMs != nil {
		cfg.RetryBackoffMs = *req.RetryBackoffMs
	}
	if req.RPM != nil {
		cfg.RPM = *req.RPM
	}

	if err := llm.SetConfig(cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	llm.Reg().EvictAll()
	Audit(c, "llm_config_put_legacy", cfg.Model)
	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"config": toConfigView(llm.CurrentConfig()),
	})
}

func toConfigView(cfg llm.Config) llmConfigView {
	apiURL := ""
	if len(cfg.ChatEndpoints) > 0 {
		apiURL = cfg.ChatEndpoints[0]
	}
	return llmConfigView{
		Provider:       string(cfg.Kind),
		APIURL:         apiURL,
		Model:          cfg.Model,
		HasAPIKey:      strings.TrimSpace(cfg.APIKey) != "",
		TimeoutSeconds: cfg.TimeoutSeconds,
		MaxRetries:     cfg.MaxRetries,
		RetryBackoffMs: cfg.RetryBackoffMs,
		RPM:            cfg.RPM,
	}
}

func profileAsConfigView(p llm.Profile) llmConfigView {
	return llmConfigView{
		Provider:       string(p.Provider),
		APIURL:         p.Endpoint,
		Model:          p.Model,
		HasAPIKey:      strings.TrimSpace(envGet(p.APIKeyRef)) != "",
		TimeoutSeconds: p.TimeoutSeconds,
		MaxRetries:     p.MaxRetries,
		RetryBackoffMs: p.RetryBackoffMs,
		RPM:            p.RPM,
	}
}
