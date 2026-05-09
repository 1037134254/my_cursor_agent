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
	{
		api.POST("/chat", ChatHandler)
		api.GET("/chat/ws", StreamChatHandler)
		api.GET("/llm/config", LLMConfigGetHandler)
		api.PUT("/llm/config", LLMConfigPutHandler)
		api.GET("/workspace/files", WorkspaceListFilesHandler)
		api.GET("/workspace/file", WorkspaceReadFileHandler)
		api.PUT("/workspace/file", WorkspaceWriteFileHandler)
		api.POST("/rag/ingest", RAGIngestHandler)
		api.POST("/rag/search", RAGSearchHandler)
		api.POST("/rag/index-code", RAGIndexCodeHandler)
	}
}
