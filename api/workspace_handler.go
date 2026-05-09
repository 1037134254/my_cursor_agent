package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"my_cursor/internal/tool"

	"github.com/gin-gonic/gin"
)

type workspaceWriteReq struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WorkspaceListFilesHandler GET /api/workspace/files — 列出当前租户工作区下文件。
func WorkspaceListFilesHandler(c *gin.Context) {
	p := PrincipalFrom(c)
	root := tool.DirForTenant(p.TenantID)
	_ = os.MkdirAll(root, 0755)
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	c.JSON(http.StatusOK, gin.H{"code": 0, "files": files})
}

// WorkspaceReadFileHandler GET /api/workspace/file?path=relative
func WorkspaceReadFileHandler(c *gin.Context) {
	p := PrincipalFrom(c)
	rel := strings.TrimSpace(c.Query("path"))
	full, ok := resolveWorkspaceFile(p.TenantID, rel)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效路径"})
		return
	}
	b, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"path":    filepath.ToSlash(rel),
		"content": string(b),
	})
}

// WorkspaceWriteFileHandler PUT /api/workspace/file
func WorkspaceWriteFileHandler(c *gin.Context) {
	p := PrincipalFrom(c)
	var req workspaceWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	full, ok := resolveWorkspaceFile(p.TenantID, req.Path)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效路径"})
		return
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(full, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	Audit(c, "workspace_write", filepath.ToSlash(strings.TrimSpace(req.Path)))
	c.JSON(http.StatusOK, gin.H{"code": 0, "path": filepath.ToSlash(strings.TrimSpace(req.Path))})
}

func resolveWorkspaceFile(tenantID, rel string) (full string, ok bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	if filepath.IsAbs(rel) {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	for _, p := range strings.Split(rel, "/") {
		if p == ".." {
			return "", false
		}
	}
	rel = filepath.FromSlash(rel)
	root := tool.DirForTenant(tenantID)
	full = filepath.Join(root, rel)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	sep := string(os.PathSeparator)
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs+sep, rootAbs+sep) {
		return "", false
	}
	return fullAbs, true
}
