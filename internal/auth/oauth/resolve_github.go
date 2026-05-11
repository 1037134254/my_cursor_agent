package oauth

import (
	"fmt"
	"os"
	"strings"

	"my_cursor/internal/auth"
	"my_cursor/internal/tenant"
)

// ResolveGitHubProfile 将 GitHub Profile 映射为内部 UserRecord（用于签发 JWT）。
//
// 映射顺序：
//  1. OAUTH_GITHUB_BINDINGS：login 或 numeric id -> 租户:角色
//  2. GITHUB_OAUTH_AUTO=true 时自动建档（默认租户/角色）
//
// 绑定示例：OAUTH_GITHUB_BINDINGS=octocat:default:admin;12345:acme:user
func ResolveGitHubProfile(p *Profile) (*auth.UserRecord, error) {
	if p == nil {
		return nil, fmt.Errorf("空 profile")
	}
	keyID := strings.TrimSpace(p.OpenID)     // numeric id
	keyLogin := strings.TrimSpace(p.UnionID) // login

	if rec := lookupGitHubBinding(keyID); rec != nil {
		return rec, nil
	}
	if keyLogin != "" && keyLogin != keyID {
		if rec := lookupGitHubBinding(keyLogin); rec != nil {
			return rec, nil
		}
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("GITHUB_OAUTH_AUTO")), "true") {
		tid := strings.TrimSpace(os.Getenv("GITHUB_OAUTH_DEFAULT_TENANT"))
		if tenant.SanitizeID(tid) == "" {
			tid = tenant.DefaultID
		}
		roles := parseRolesEnv(os.Getenv("GITHUB_OAUTH_DEFAULT_ROLES"))
		if len(roles) == 0 {
			roles = []string{auth.RoleUser}
		}
		sub := keyLogin
		if sub == "" {
			sub = keyID
		}
		return &auth.UserRecord{
			Username: "gh:" + sub,
			Password: "",
			TenantID: tid,
			Roles:    roles,
		}, nil
	}

	return nil, fmt.Errorf("GitHub 账号未绑定（配置 OAUTH_GITHUB_BINDINGS 或将 GITHUB_OAUTH_AUTO=true）")
}

func lookupGitHubBinding(key string) *auth.UserRecord {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	raw := strings.TrimSpace(os.Getenv("OAUTH_GITHUB_BINDINGS"))
	for _, seg := range strings.Split(raw, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		parts := strings.SplitN(seg, ":", 3)
		if len(parts) != 3 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		tid := strings.TrimSpace(parts[1])
		rolesStr := strings.TrimSpace(parts[2])
		if !strings.EqualFold(k, key) || tenant.SanitizeID(tid) == "" {
			continue
		}
		roles := parseRolesList(rolesStr)
		if len(roles) == 0 {
			roles = []string{auth.RoleUser}
		}
		cp := auth.UserRecord{
			Username: "gh:" + key,
			Password: "",
			TenantID: tid,
			Roles:    roles,
		}
		return &cp
	}
	return nil
}
