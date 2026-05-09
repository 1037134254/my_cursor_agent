// Package oauth 抽象第三方登录；当前实现微信开放平台「网站应用扫码」，后续可注册更多 Provider。
package oauth

import "context"

// Profile 第三方返回的统一身份信息。
type Profile struct {
	Provider string
	OpenID   string
	UnionID  string
	Nickname string
}

// Provider 第三方登录提供者（微信、钉钉等）。
type Provider interface {
	ID() string
	DisplayName() string
	// AuthorizationURL 跳转授权页完整 URL（含 state）。
	AuthorizationURL(state string) string
	// Exchange 用回调 code 换取 Profile。
	Exchange(ctx context.Context, code string) (*Profile, error)
}
