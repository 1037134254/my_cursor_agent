package tool

import (
	"os"
	"path/filepath"

	"my_cursor/internal/tenant"
)

const ROOT = "./workspace"

// DirForTenant 返回租户根目录下的绝对路径 workspace/<tenant>/。
func DirForTenant(tenantID string) string {
	tid := tenant.SanitizeID(tenantID)
	if tid == "" {
		tid = tenant.DefaultID
	}
	return filepath.Join(ROOT, tid)
}

// ReadFile 在指定租户工作区内读取相对路径文件。
func ReadFile(tenantID, rel string) (string, error) {
	root := DirForTenant(tenantID)
	full := filepath.Join(root, rel)
	b, err := os.ReadFile(full)
	return string(b), err
}

// WriteFile 在指定租户工作区内写入文件。
func WriteFile(tenantID, path, content string) error {
	root := DirForTenant(tenantID)
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0644)
}
