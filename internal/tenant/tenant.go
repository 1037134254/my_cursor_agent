package tenant

import (
	"regexp"
	"strings"
)

var safeTenant = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// SanitizeID 将租户 id 规范为安全目录名；非法则返回空。
func SanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || !safeTenant.MatchString(id) {
		return ""
	}
	return id
}

// DefaultID 未认证或调试模式下使用的租户。
const DefaultID = "default"
