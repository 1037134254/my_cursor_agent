package api

import (
	"net/http"
	"strings"

	"my_cursor/internal/llm"

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

// LLMConfigGetHandler 返回当前运行时配置（隐藏密钥原文）。
func LLMConfigGetHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"config": toConfigView(llm.CurrentConfig()),
	})
}

// LLMConfigPutHandler 更新运行时配置并立即生效（无需重启服务）。
func LLMConfigPutHandler(c *gin.Context) {
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
	Audit(c, "llm_config_put", cfg.Model)
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
