package api

import (
	"net/http"
	"os"
	"strings"

	"my_cursor/internal/codeindex"
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

type ragIndexCodeRequest struct {
	Root     string `json:"root"`
	MaxLines int    `json:"max_lines"`
	Overlap  int    `json:"overlap"`
	Exts     string `json:"exts"`
}

// RAGIndexCodeHandler 扫描仓库源码，按行切片并写入向量库（需 Qdrant + 嵌入模型）。
func RAGIndexCodeHandler(c *gin.Context) {
	var req ragIndexCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	root := strings.TrimSpace(req.Root)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("CODE_INDEX_ROOT"))
	}
	if root == "" {
		root = "."
	}
	maxLines := req.MaxLines
	if maxLines <= 0 {
		maxLines = 120
	}
	overlap := req.Overlap
	if overlap < 0 {
		overlap = 0
	}
	exts := codeindex.ParseExtList(req.Exts)
	if len(exts) == 0 {
		exts = codeindex.ParseExtList(os.Getenv("CODE_INDEX_EXTS"))
	}

	files, chunks, err := codeindex.IndexCodeRoot(c.Request.Context(), root, maxLines, overlap, exts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"files_index": files,
		"chunks":      chunks,
		"root":        root,
	})
}
