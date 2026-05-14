package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Static("/assets", "./web/dist/assets")
	r.Static("/web", "./web")

	r.GET("/", func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	api := r.Group("/api")
	api.POST("/auth/login", LoginHandler)
	api.POST("/auth/refresh", RefreshHandler)
	api.POST("/auth/logout", LogoutHandler)

	api.GET("/auth/oauth/providers", OAuthProvidersHandler)
	api.GET("/auth/oauth/wechat/start", OAuthWeChatStartHandler)
	api.GET("/auth/oauth/wechat/callback", OAuthWeChatCallbackHandler)
	api.GET("/auth/oauth/github/start", OAuthGitHubStartHandler)
	api.GET("/auth/oauth/github/callback", OAuthGitHubCallbackHandler)

	// WebSocket 在 Handler 内单独鉴权（支持 query access_token）
	api.GET("/chat/ws", StreamChatHandler)

	sec := api.Group("")
	sec.Use(AuthMiddleware())
	sec.POST("/chat", RequirePermission(PermChat), ChatHandler)

	// 旧 LLM 配置（已废弃，仅作兼容；会写到 default 租户 default chat profile）。
	sec.GET("/llm/config", RequirePermission(PermLLMRead), LLMConfigGetHandler)
	sec.PUT("/llm/config", RequirePermission(PermLLMWrite), LLMConfigPutHandler)

	// 新模型注册表：profile 全局管理（admin only），租户级查询（本租户用户可读）。
	sec.GET("/llm/profiles", RequirePermission(PermLLMAdmin), LLMProfilesListHandler)
	sec.POST("/llm/profiles", RequirePermission(PermLLMAdmin), LLMProfileUpsertHandler)
	sec.PUT("/llm/profiles/:id", RequirePermission(PermLLMAdmin), LLMProfileUpsertHandler)
	sec.DELETE("/llm/profiles/:id", RequirePermission(PermLLMAdmin), LLMProfileDeleteHandler)
	sec.POST("/llm/profiles/:id/probe", RequirePermission(PermLLMAdmin), LLMProfileProbeHandler)
	sec.GET("/llm/current", RequirePermission(PermLLMRead), LLMCurrentHandler)
	sec.GET("/tenants/:tid/models", RequirePermission(PermLLMRead), TenantModelsListHandler)
	sec.PUT("/tenants/:tid/models", RequirePermission(PermLLMAdmin), TenantModelsPutHandler)

	sec.GET("/workspace/files", RequirePermission(PermWorkspaceRead), WorkspaceListFilesHandler)
	sec.GET("/workspace/file", RequirePermission(PermWorkspaceRead), WorkspaceReadFileHandler)
	sec.PUT("/workspace/file", RequirePermission(PermWorkspaceWrite), WorkspaceWriteFileHandler)
	sec.POST("/rag/ingest", RequirePermission(PermRAGWrite), RAGIngestHandler)
	sec.POST("/rag/search", RequirePermission(PermRAGSearch), RAGSearchHandler)
	sec.POST("/rag/index-code", RequirePermission(PermRAGWrite), RAGIndexCodeHandler)
}
