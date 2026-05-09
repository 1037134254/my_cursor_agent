package api

import (
	"net/http"

	"my_cursor/internal/rag"

	"github.com/gin-gonic/gin"
)

type ragIngestRequest struct {
	Text     string `json:"text"`
	Source   string `json:"source"`
	MaxRunes int    `json:"max_runes"`
	Overlap  int    `json:"overlap"`
}

func RAGIngestHandler(c *gin.Context) {
	var req ragIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	svc, err := rag.Get()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	n, err := svc.Ingest(c.Request.Context(), req.Text, req.Source, req.MaxRunes, req.Overlap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "chunks": n})
}

type ragSearchRequest struct {
	Query string `json:"query"`
	Limit uint64 `json:"limit"`
}

func RAGSearchHandler(c *gin.Context) {
	var req ragSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	svc, err := rag.Get()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	hits, err := svc.Search(c.Request.Context(), req.Query, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": hits})
}
