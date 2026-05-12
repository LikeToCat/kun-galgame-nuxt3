package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kun-galgame-api/pkg/config"
)

// oauthHTTPTimeout caps every OAuth-server roundtrip (token exchange /
// refresh / revoke / userinfo) at 10s. Without this every login + every
// authenticated request would block indefinitely if the OAuth server hung,
// since the four hot paths all run synchronously in the request hot path.
const oauthHTTPTimeout = 10 * time.Second

// Client calls the OAuth server via HTTP.
// It is a thin transport layer: it performs raw HTTP calls and decodes the
// standard {code, message, data} wrapper used by the OAuth server. No
// business logic lives here.
type Client struct {
	cfg        config.OAuthConfig
	httpClient *http.Client
}

// NewClient constructs an OAuth HTTP client with the given configuration.
// The HTTP client carries a per-request timeout so a hung OAuth server can't
// stall login / token refresh / logout indefinitely.
func NewClient(cfg config.OAuthConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: oauthHTTPTimeout},
	}
}

// TokenResponse represents the token data inside the OAuth response wrapper.
// /oauth/token returns { code: 0, message: "成功", data: { access_token, ... } }
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// UserInfo represents the OAuth userinfo payload.
//
// IMPORTANT: kungal post-migration relies on the integer `id` (= OAuth
// users.id) and the `roles` array. The OIDC userinfo standard only
// requires `sub` (UUID). The OAuth team must extend /oauth/userinfo to
// include `id` and `roles` so kungal can derive its uid + admin role
// without a second round-trip.
type UserInfo struct {
	ID        int      `json:"id"`
	Sub       string   `json:"sub"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Picture   string   `json:"picture"`
	Roles     []string `json:"roles"`
	UpdatedAt int64    `json:"updated_at"`
}

// ExchangeCode exchanges an authorization code for access/refresh tokens.
// NOTE: /oauth/token returns a wrapped { code, message, data } response.
func (c *Client) ExchangeCode(code, codeVerifier string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  c.cfg.RedirectURI,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
		"code_verifier": codeVerifier,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化 token 请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", c.cfg.ServerURL+"/oauth/token", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OAuth token 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OAuth token 请求失败, 状态码: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 token 响应失败: %w", err)
	}

	// /oauth/token returns { code: 0, message: "成功", data: { access_token, ... } }
	var wrapper struct {
		Code int            `json:"code"`
		Data *TokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 token 响应失败: %w, body: %s", err, string(respBody))
	}
	if wrapper.Code != 0 || wrapper.Data == nil {
		return nil, fmt.Errorf("token 交换失败: code=%d, body: %s", wrapper.Code, string(respBody))
	}
	if wrapper.Data.AccessToken == "" {
		return nil, fmt.Errorf("token 响应无 access_token, body: %s", string(respBody))
	}
	return wrapper.Data, nil
}

// FetchUserInfo retrieves the OAuth user info using an access token.
// NOTE: /oauth/userinfo returns the wrapped { code, message, data } response.
func (c *Client) FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.cfg.ServerURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("创建 userinfo 请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 userinfo 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo 请求失败, 状态码: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 userinfo 响应失败: %w", err)
	}

	// /oauth/userinfo returns { code: 0, message: "成功", data: { sub, name, ... } }
	var wrapper struct {
		Code int       `json:"code"`
		Data *UserInfo `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 userinfo 响应失败: %w, body: %s", err, string(respBody))
	}
	if wrapper.Code != 0 || wrapper.Data == nil {
		return nil, fmt.Errorf("userinfo 返回错误: code=%d, body: %s", wrapper.Code, string(respBody))
	}
	return wrapper.Data, nil
}

// RevokeToken revokes a refresh token against the OAuth server.
func (c *Client) RevokeToken(refreshToken string) error {
	payload, err := json.Marshal(map[string]string{"token": refreshToken})
	if err != nil {
		return fmt.Errorf("序列化 revoke 请求失败: %w", err)
	}
	req, err := http.NewRequest("POST", c.cfg.ServerURL+"/oauth/revoke", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建 revoke 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RefreshOAuthToken refreshes the OAuth tokens using the refresh token.
func (c *Client) RefreshOAuthToken(refreshToken string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化刷新请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", c.cfg.ServerURL+"/oauth/token", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建刷新请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("刷新 token 失败, 状态码: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取刷新响应失败: %w", err)
	}

	var wrapper struct {
		Code int            `json:"code"`
		Data *TokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}
	if wrapper.Code != 0 || wrapper.Data == nil || wrapper.Data.AccessToken == "" {
		return nil, fmt.Errorf("刷新 token 失败: %s", string(respBody))
	}
	return wrapper.Data, nil
}
