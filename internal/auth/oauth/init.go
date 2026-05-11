package oauth

// InitProviders 根据环境变量注册各 OAuth Provider。
// 当前支持：微信开放平台（企业）TODO
// GitHub OAuth（个人即可）。
func InitProviders() {
	regMu.Lock()
	providers = make(map[string]Provider)
	displayList = nil
	regMu.Unlock()

	if w := NewWeChatWebFromEnv(); w.Configured() {
		RegisterProvider(w)
	}
	if g := NewGitHubFromEnv(); g.Configured() {
		RegisterProvider(g)
	}
}
