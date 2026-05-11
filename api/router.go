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
	sec.GET("/llm/config", RequirePermission(PermLLMRead), LLMConfigGetHandler)
	sec.PUT("/llm/config", RequirePermission(PermLLMWrite), LLMConfigPutHandler)
	sec.GET("/workspace/files", RequirePermission(PermWorkspaceRead), WorkspaceListFilesHandler)
	sec.GET("/workspace/file", RequirePermission(PermWorkspaceRead), WorkspaceReadFileHandler)
	sec.PUT("/workspace/file", RequirePermission(PermWorkspaceWrite), WorkspaceWriteFileHandler)
	sec.POST("/rag/ingest", RequirePermission(PermRAGWrite), RAGIngestHandler)
	sec.POST("/rag/search", RequirePermission(PermRAGSearch), RAGSearchHandler)
	sec.POST("/rag/index-code", RequirePermission(PermRAGWrite), RAGIndexCodeHandler)
}
