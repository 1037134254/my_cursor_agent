package oauth

import (
	"sync"
)

var (
	regMu       sync.RWMutex
	providers   = map[string]Provider{}
	displayList []providerMeta
)

type providerMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RegisterProvider 注册第三方登录（进程启动时调用）。
func RegisterProvider(p Provider) {
	if p == nil || p.ID() == "" {
		return
	}
	regMu.Lock()
	defer regMu.Unlock()
	providers[p.ID()] = p
	displayList = append(displayList, providerMeta{ID: p.ID(), Name: p.DisplayName()})
}

// ProviderByID 查找已注册 Provider。
func ProviderByID(id string) (Provider, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	p, ok := providers[id]
	return p, ok
}

// ListProviders 返回已注册列表（供前端展示「更多登录方式」）。
func ListProviders() []providerMeta {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]providerMeta, len(displayList))
	copy(out, displayList)
	return out
}
