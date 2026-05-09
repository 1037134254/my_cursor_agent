package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Static("/web", "./web")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	api := r.Group("/api")
	{
		api.POST("/chat", ChatHandler)
		api.GET("/chat/ws", StreamChatHandler)
		api.POST("/rag/ingest", RAGIngestHandler)
		api.POST("/rag/search", RAGSearchHandler)
		api.POST("/rag/index-code", RAGIndexCodeHandler)
	}
}
