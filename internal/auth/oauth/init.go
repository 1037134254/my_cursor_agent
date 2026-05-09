package oauth

// InitProviders 根据环境变量注册各 OAuth Provider（微信等）。
func InitProviders() {
	regMu.Lock()
	providers = make(map[string]Provider)
	displayList = nil
	regMu.Unlock()

	w := NewWeChatWebFromEnv()
	if w.Configured() {
		RegisterProvider(w)
	}
}
