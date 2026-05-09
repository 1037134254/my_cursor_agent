package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ProviderIDWeChatWeb 微信网站应用扫码（与 RegisterProvider 注册的 ID 一致）。
const ProviderIDWeChatWeb = "wechat_web"

// WeChatWebProvider 微信开放平台 — 网站应用扫码登录（qrconnect）。
// 文档：https://developers.weixin.qq.com/doc/oplatform/Website_App/WeChat_Login/Wechat_Login.html
type WeChatWebProvider struct {
	AppID       string
	AppSecret   string
	RedirectURI string
}

func NewWeChatWebFromEnv() *WeChatWebProvider {
	return &WeChatWebProvider{
		AppID:       strings.TrimSpace(os.Getenv("WECHAT_OPEN_APP_ID")),
		AppSecret:   strings.TrimSpace(os.Getenv("WECHAT_OPEN_APP_SECRET")),
		RedirectURI: strings.TrimSpace(os.Getenv("WECHAT_REDIRECT_URI")),
	}
}

func (p *WeChatWebProvider) Configured() bool {
	return p.AppID != "" && p.AppSecret != "" && p.RedirectURI != ""
}

func (p *WeChatWebProvider) ID() string { return ProviderIDWeChatWeb }

func (p *WeChatWebProvider) DisplayName() string { return "微信扫码" }

func (p *WeChatWebProvider) AuthorizationURL(state string) string {
	v := url.Values{}
	v.Set("appid", p.AppID)
	v.Set("redirect_uri", p.RedirectURI)
	v.Set("response_type", "code")
	v.Set("scope", "snsapi_login")
	v.Set("state", state)
	return "https://open.weixin.qq.com/connect/qrconnect?" + v.Encode() + "#wechat_redirect"
}

type wechatTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfo struct {
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	UnionID  string `json:"unionid"`
	ErrCode  int    `json:"errcode"`
	ErrMsg   string `json:"errmsg"`
}

func (p *WeChatWebProvider) Exchange(ctx context.Context, code string) (*Profile, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("微信登录未配置 WECHAT_OPEN_APP_ID/SECRET/WECHAT_REDIRECT_URI")
	}
	u, err := url.Parse("https://api.weixin.qq.com/sns/oauth2/access_token")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("appid", p.AppID)
	q.Set("secret", p.AppSecret)
	q.Set("code", strings.TrimSpace(code))
	q.Set("grant_type", "authorization_code")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tr wechatTokenResp
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("解析 token 响应: %w body=%s", err, truncate(body, 200))
	}
	if tr.ErrCode != 0 {
		return nil, fmt.Errorf("微信 token 错误: %d %s", tr.ErrCode, tr.ErrMsg)
	}
	if tr.AccessToken == "" || tr.OpenID == "" {
		return nil, fmt.Errorf("微信返回缺少 access_token/openid: %s", truncate(body, 300))
	}

	prof := &Profile{
		Provider: ProviderIDWeChatWeb,
		OpenID:   tr.OpenID,
		UnionID:  tr.UnionID,
	}

	// 补充昵称（失败则忽略）
	ui := wechatUserInfo{}
	userURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		url.QueryEscape(tr.AccessToken), url.QueryEscape(tr.OpenID))
	if req2, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil); err == nil {
		if resp2, err := httpClient.Do(req2); err == nil {
			func() {
				defer func() { _ = resp2.Body.Close() }()
				b2, err := io.ReadAll(resp2.Body)
				if err != nil {
					return
				}
				_ = json.Unmarshal(b2, &ui)
			}()
			if ui.ErrCode == 0 && ui.Nickname != "" {
				prof.Nickname = ui.Nickname
			}
			if prof.UnionID == "" && ui.UnionID != "" {
				prof.UnionID = ui.UnionID
			}
		}
	}

	return prof, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
