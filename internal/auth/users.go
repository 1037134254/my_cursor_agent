package auth

import (
	"os"
	"strings"

	"my_cursor/internal/tenant"
)

type UserRecord struct {
	Username string
	Password string
	TenantID string
	Roles    []string
}

// LoadUsers 从 AUTH_USERS 加载账号。
// 格式（多条用英文分号分隔）：username:password:tenant_id:role[,role2];...
// password 中勿包含冒号；租户须符合 tenant.SanitizeID。
func LoadUsers() []UserRecord {
	raw := strings.TrimSpace(os.Getenv("AUTH_USERS"))
	if raw == "" {
		return nil
	}
	var out []UserRecord
	for _, seg := range strings.Split(raw, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		parts := strings.SplitN(seg, ":", 4)
		if len(parts) != 4 {
			continue
		}
		u, pass, tid, rolesStr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), strings.TrimSpace(parts[3])
		if u == "" || pass == "" || tenant.SanitizeID(tid) == "" {
			continue
		}
		var roles []string
		for _, r := range strings.Split(rolesStr, ",") {
			r = strings.TrimSpace(strings.ToLower(r))
			if r != "" {
				roles = append(roles, r)
			}
		}
		if len(roles) == 0 {
			roles = []string{RoleUser}
		}
		out = append(out, UserRecord{
			Username: u,
			Password: pass,
			TenantID: tid,
			Roles:    roles,
		})
	}
	return out
}

func FindUser(username, password string) *UserRecord {
	for _, u := range LoadUsers() {
		if u.Username == username && u.Password == password {
			cp := u
			return &cp
		}
	}
	return nil
}
