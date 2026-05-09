package oauth

import (
	"fmt"
	"os"
	"strings"

	"my_cursor/internal/auth"
	"my_cursor/internal/tenant"
)

// ResolveWeChatProfile 将微信 Profile 映射为内部 UserRecord（用于签发 JWT）。
//
// 映射顺序：
//  1. 环境变量 OAUTH_WECHAT_BINDINGS（unionid 或 openid -> 租户:角色）
//  2. WECHAT_OAUTH_AUTO=true 时自动建档（默认租户/角色）
func ResolveWeChatProfile(p *Profile) (*auth.UserRecord, error) {
	if p == nil {
		return nil, fmt.Errorf("空 profile")
	}
	keyUnion := strings.TrimSpace(p.UnionID)
	keyOpen := strings.TrimSpace(p.OpenID)

	if rec := lookupBinding(keyUnion); rec != nil {
		return rec, nil
	}
	if keyUnion != keyOpen {
		if rec := lookupBinding(keyOpen); rec != nil {
			return rec, nil
		}
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("WECHAT_OAUTH_AUTO")), "true") {
		tid := strings.TrimSpace(os.Getenv("WECHAT_OAUTH_DEFAULT_TENANT"))
		if tenant.SanitizeID(tid) == "" {
			tid = tenant.DefaultID
		}
		roles := parseRolesEnv(os.Getenv("WECHAT_OAUTH_DEFAULT_ROLES"))
		if len(roles) == 0 {
			roles = []string{auth.RoleUser}
		}
		sub := keyUnion
		if sub == "" {
			sub = keyOpen
		}
		return &auth.UserRecord{
			Username: "wx:" + sub,
			Password: "",
			TenantID: tid,
			Roles:    roles,
		}, nil
	}

	return nil, fmt.Errorf("微信账号未绑定（配置 OAUTH_WECHAT_BINDINGS 或将 WECHAT_OAUTH_AUTO=true）")
}

// OAUTH_WECHAT_BINDINGS： unionid或openid:租户id:角色1,角色2 ; 多条用英文分号分隔。
func lookupBinding(key string) *auth.UserRecord {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	raw := strings.TrimSpace(os.Getenv("OAUTH_WECHAT_BINDINGS"))
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
		if k != key || tenant.SanitizeID(tid) == "" {
			continue
		}
		roles := parseRolesList(rolesStr)
		if len(roles) == 0 {
			roles = []string{auth.RoleUser}
		}
		cp := auth.UserRecord{
			Username: "wx:" + key,
			Password: "",
			TenantID: tid,
			Roles:    roles,
		}
		return &cp
	}
	return nil
}

func parseRolesEnv(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return parseRolesList(s)
}

func parseRolesList(s string) []string {
	var roles []string
	for _, r := range strings.Split(s, ",") {
		r = strings.TrimSpace(strings.ToLower(r))
		if r != "" {
			roles = append(roles, r)
		}
	}
	return roles
}
