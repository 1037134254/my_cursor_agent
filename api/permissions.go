package api

import (
	"my_cursor/internal/auth"
)

const (
	PermChat           = "chat"
	PermWorkspaceRead  = "workspace_read"
	PermWorkspaceWrite = "workspace_write"
	PermRAGSearch      = "rag_search"
	PermRAGWrite       = "rag_write"
	PermLLMRead        = "llm_read"
	PermLLMWrite       = "llm_write"
	PermLLMAdmin       = "llm_admin" // 管 profile / 租户绑定，admin only
)

// Can 校验 Principal 是否具备某权限（AUTH_ENABLED=false 时恒为 true）。
func Can(p auth.Principal, perm string) bool {
	if !auth.Enabled() {
		return true
	}
	switch perm {
	case PermChat:
		return p.HasAny(auth.RoleAnon, auth.RoleViewer, auth.RoleUser, auth.RoleAdmin)
	case PermWorkspaceRead:
		return p.HasAny(auth.RoleViewer, auth.RoleUser, auth.RoleAdmin)
	case PermWorkspaceWrite:
		return p.HasAny(auth.RoleUser, auth.RoleAdmin)
	case PermRAGSearch:
		return p.HasAny(auth.RoleViewer, auth.RoleUser, auth.RoleAdmin)
	case PermRAGWrite:
		return p.HasAny(auth.RoleUser, auth.RoleAdmin)
	case PermLLMRead:
		return p.HasAny(auth.RoleViewer, auth.RoleUser, auth.RoleAdmin)
	case PermLLMWrite:
		return p.HasRole(auth.RoleAdmin)
	case PermLLMAdmin:
		return p.HasRole(auth.RoleAdmin)
	default:
		return false
	}
}
