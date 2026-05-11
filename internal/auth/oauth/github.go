package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProviderIDGitHub 第三方 ID，与前端按钮渲染保持一致。
const ProviderIDGitHub = "github"

// GitHubProvider GitHub OAuth App（开发者首选；国内可用，5 分钟申请）。
// 申请入口：https://github.com/settings/developers → New OAuth App
//   - Authorization callback URL 填到 GITHUB_REDIRECT_URI（必须完全一致）
type GitHubProvider struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

func NewGitHubFromEnv() *GitHubProvider {
	return &GitHubProvider{
		ClientID:     strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
		RedirectURI:  strings.TrimSpace(os.Getenv("GITHUB_REDIRECT_URI")),
	}
}

func (p *GitHubProvider) Configured() bool {
	return p.ClientID != "" && p.ClientSecret != "" && p.RedirectURI != ""
}

func (p *GitHubProvider) ID() string { return ProviderIDGitHub }

func (p *GitHubProvider) DisplayName() string { return "GitHub 登录" }

func (p *GitHubProvider) AuthorizationURL(state string) string {
	v := url.Values{}
	v.Set("client_id", p.ClientID)
	v.Set("redirect_uri", p.RedirectURI)
	v.Set("scope", "read:user user:email")
	v.Set("state", state)
	v.Set("allow_signup", "false")
	return "https://github.com/login/oauth/authorize?" + v.Encode()
}

type githubTokenResp struct {
	AccessToken      string `json:"access_token"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type githubUserInfo struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*Profile, error) {
	if !p.Configured() {
		return nil, fmt.Errorf("GitHub 登录未配置 GITHUB_CLIENT_ID/SECRET/REDIRECT_URI")
	}
	httpClient := &http.Client{Timeout: 15 * time.Second}

	form := url.Values{}
	form.Set("client_id", p.ClientID)
	form.Set("client_secret", p.ClientSecret)
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", p.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "my_cursor/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tr githubTokenResp
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("解析 token 响应: %w body=%s", err, truncate(body, 200))
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("GitHub token 错误: %s %s", tr.Error, tr.ErrorDescription)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("GitHub 返回空 access_token: %s", truncate(body, 300))
	}

	user, err := p.fetchUser(ctx, httpClient, tr.AccessToken)
	if err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, fmt.Errorf("GitHub 用户信息为空")
	}

	nickname := strings.TrimSpace(user.Name)
	if nickname == "" {
		nickname = user.Login
	}
	return &Profile{
		Provider: ProviderIDGitHub,
		// 数字 ID 永久稳定；Username（login）可能改名所以放 UnionID
		OpenID:   strconv.FormatInt(user.ID, 10),
		UnionID:  user.Login,
		Nickname: nickname,
	}, nil
}

func (p *GitHubProvider) fetchUser(ctx context.Context, c *http.Client, token string) (*githubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "my_cursor/1.0")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub /user 状态 %d body=%s", resp.StatusCode, truncate(body, 200))
	}
	var u githubUserInfo
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("解析 GitHub user: %w", err)
	}
	return &u, nil
}
