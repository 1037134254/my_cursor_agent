package api

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"my_cursor/internal/auth"
	"my_cursor/internal/auth/oauth"

	"github.com/gin-gonic/gin"
)

// OAuthProvidersHandler GET /api/auth/oauth/providers — 已注册的第三方登录（便于前端渲染按钮）。
func OAuthProvidersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "providers": oauth.ListProviders()})
}

// OAuthWeChatStartHandler GET /api/auth/oauth/wechat/start — 返回微信 qrconnect 跳转 URL。
func OAuthWeChatStartHandler(c *gin.Context) {
	if !auth.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先启用 AUTH_ENABLED 并配置 JWT_SECRET"})
		return
	}
	p, ok := oauth.ProviderByID(oauth.ProviderIDWeChatWeb)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "微信登录未配置 WECHAT_OPEN_APP_ID / WECHAT_OPEN_APP_SECRET / WECHAT_REDIRECT_URI"})
		return
	}
	state, err := oauth.NewState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	authURL := p.AuthorizationURL(state)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"url":   authURL,
		"state": state,
	})
}

// OAuthWeChatCallbackHandler GET /api/auth/oauth/wechat/callback — 微信回调，换票后重定向回前端并附带 token。
func OAuthWeChatCallbackHandler(c *gin.Context) {
	if !auth.Enabled() {
		redirectOAuthResult(c, false, "auth_disabled", "", "")
		return
	}
	state := strings.TrimSpace(c.Query("state"))
	if !oauth.ConsumeState(state) {
		redirectOAuthResult(c, false, "invalid_state", "", "")
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		redirectOAuthResult(c, false, "missing_code", "", "")
		return
	}
	w := oauth.NewWeChatWebFromEnv()
	prof, err := w.Exchange(c.Request.Context(), code)
	if err != nil {
		log.Printf("oauth wechat exchange failed ip=%s err=%v", c.ClientIP(), err)
		redirectOAuthResult(c, false, "exchange_failed", "", "")
		return
	}
	rec, err := oauth.ResolveWeChatProfile(prof)
	if err != nil {
		log.Printf("oauth wechat unbound ip=%s err=%v", c.ClientIP(), err)
		redirectOAuthResult(c, false, "not_bound", "", "")
		return
	}
	access, err := auth.IssueAccessJWT(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	refresh, err := auth.NewRefreshToken(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("oauth wechat ok user=%s tenant=%s ip=%s", rec.Username, rec.TenantID, c.ClientIP())
	redirectOAuthResult(c, true, "", access, refresh)
}

// OAuthGitHubStartHandler GET /api/auth/oauth/github/start — 返回 GitHub 授权页 URL。
func OAuthGitHubStartHandler(c *gin.Context) {
	if !auth.Enabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先启用 AUTH_ENABLED 并配置 JWT_SECRET"})
		return
	}
	p, ok := oauth.ProviderByID(oauth.ProviderIDGitHub)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub 登录未配置 GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET / GITHUB_REDIRECT_URI"})
		return
	}
	state, err := oauth.NewState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	authURL := p.AuthorizationURL(state)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"url":   authURL,
		"state": state,
	})
}

// OAuthGitHubCallbackHandler GET /api/auth/oauth/github/callback — GitHub 回调，换票后重定向回前端并附带 token。
func OAuthGitHubCallbackHandler(c *gin.Context) {
	if !auth.Enabled() {
		redirectOAuthResult(c, false, "auth_disabled", "", "")
		return
	}
	state := strings.TrimSpace(c.Query("state"))
	if !oauth.ConsumeState(state) {
		redirectOAuthResult(c, false, "invalid_state", "", "")
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		redirectOAuthResult(c, false, "missing_code", "", "")
		return
	}
	g := oauth.NewGitHubFromEnv()
	prof, err := g.Exchange(c.Request.Context(), code)
	if err != nil {
		log.Printf("oauth github exchange failed ip=%s err=%v", c.ClientIP(), err)
		redirectOAuthResult(c, false, "exchange_failed", "", "")
		return
	}
	rec, err := oauth.ResolveGitHubProfile(prof)
	if err != nil {
		log.Printf("oauth github unbound ip=%s err=%v", c.ClientIP(), err)
		redirectOAuthResult(c, false, "not_bound", "", "")
		return
	}
	access, err := auth.IssueAccessJWT(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	refresh, err := auth.NewRefreshToken(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("oauth github ok user=%s tenant=%s ip=%s", rec.Username, rec.TenantID, c.ClientIP())
	redirectOAuthResult(c, true, "", access, refresh)
}

func redirectOAuthResult(c *gin.Context, ok bool, errCode, access, refresh string) {
	// 通用回调地址：优先 OAUTH_AFTER_LOGIN_REDIRECT，回退 WECHAT_AFTER_LOGIN_REDIRECT，再回退根 "/"
	base := strings.TrimSpace(os.Getenv("OAUTH_AFTER_LOGIN_REDIRECT"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("WECHAT_AFTER_LOGIN_REDIRECT"))
	}
	if base == "" {
		base = "/"
	}
	if !ok {
		u, err := url.Parse(base)
		if err != nil {
			c.Redirect(http.StatusFound, "/?oauth_error="+url.QueryEscape(errCode))
			return
		}
		q := u.Query()
		q.Set("oauth_error", errCode)
		u.RawQuery = q.Encode()
		c.Redirect(http.StatusFound, u.String())
		return
	}
	u, err := url.Parse(base)
	if err != nil {
		c.Redirect(http.StatusFound, "/?access_token="+url.QueryEscape(access)+"&refresh_token="+url.QueryEscape(refresh))
		return
	}
	q := u.Query()
	q.Set("access_token", access)
	q.Set("refresh_token", refresh)
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}
