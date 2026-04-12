package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/model"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.UserRepository
	rdb      *redis.Client
	oauthCfg config.OAuthConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	rdb *redis.Client,
	oauthCfg config.OAuthConfig,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		rdb:      rdb,
		oauthCfg: oauthCfg,
	}
}

// OAuthCallback exchanges the authorization code for tokens,
// fetches user info, finds or creates the local user, and creates a session.
func (s *AuthService) OAuthCallback(ctx context.Context, req *dto.OAuthCallbackRequest) (*dto.SessionResponse, *errors.AppError) {
	// 1. Exchange code for tokens
	// NOTE: /oauth/token returns raw OAuth format (NOT wrapped in {code, data})
	tokenResp, err := s.exchangeCode(req.Code, req.CodeVerifier)
	if err != nil {
		return nil, errors.ErrBadRequest(fmt.Sprintf("OAuth 授权码交换失败: %v", err))
	}

	// 2. Fetch user info from OAuth server
	// NOTE: /oauth/userinfo returns wrapped format {code, data: {...}}
	oauthUser, err := s.fetchUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, errors.ErrBadRequest(fmt.Sprintf("获取 OAuth 用户信息失败: %v", err))
	}

	// 3. Find or create local user
	user, appErr := s.findOrCreateUser(oauthUser)
	if appErr != nil {
		return nil, appErr
	}

	// 4. Create session in Redis
	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, errors.ErrInternal("生成会话令牌失败")
	}

	sessionData := middleware.SessionData{
		UserInfo: middleware.UserInfo{
			UID:   user.ID,
			Sub:   oauthUser.Sub,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
		OAuthAccessToken:  tokenResp.AccessToken,
		OAuthRefreshToken: tokenResp.RefreshToken,
		OAuthExpiresAt:    time.Now().Unix() + int64(tokenResp.ExpiresIn),
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return nil, errors.ErrInternal("序列化会话数据失败")
	}
	s.rdb.Set(ctx, "session:"+sessionToken, data, 7*24*time.Hour)

	return &dto.SessionResponse{
		Token: sessionToken,
		User: &dto.UserProfile{
			ID:          user.ID,
			Name:        user.Name,
			Email:       user.Email,
			Avatar:      user.Avatar,
			Role:        user.Role,
			Moemoepoint: user.Moemoepoint,
			Bio:         user.Bio,
		},
	}, nil
}

// Logout deletes the session from Redis and revokes the OAuth token.
func (s *AuthService) Logout(ctx context.Context, sessionToken string) error {
	val, err := s.rdb.Get(ctx, "session:"+sessionToken).Result()
	if err == nil {
		var session middleware.SessionData
		if json.Unmarshal([]byte(val), &session) == nil && session.OAuthRefreshToken != "" {
			_ = s.revokeToken(session.OAuthRefreshToken)
		}
	}
	return s.rdb.Del(ctx, "session:"+sessionToken).Err()
}

// GetProfile returns a user's full profile by ID.
func (s *AuthService) GetProfile(ctx context.Context, userID int) (*dto.UserProfile, *errors.AppError) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.ErrNotFound("用户不存在")
	}
	return &dto.UserProfile{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Avatar:      user.Avatar,
		Role:        user.Role,
		Moemoepoint: user.Moemoepoint,
		Bio:         user.Bio,
	}, nil
}

// ──────────────────────────────────────────
// OAuth HTTP helpers
// ──────────────────────────────────────────

// oauthTokenResponse is the raw OAuth 2.0 token response.
// /oauth/token returns this directly (NOT wrapped in {code, data}).
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type oauthUserInfo struct {
	Sub       string `json:"sub"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Picture   string `json:"picture"`
	UpdatedAt int64  `json:"updated_at"`
}

func (s *AuthService) exchangeCode(code, codeVerifier string) (*oauthTokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  s.oauthCfg.RedirectURI,
		"client_id":     s.oauthCfg.ClientID,
		"client_secret": s.oauthCfg.ClientSecret,
		"code_verifier": codeVerifier,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化 token 请求失败: %w", err)
	}

	resp, err := http.Post(
		s.oauthCfg.ServerURL+"/oauth/token",
		"application/json",
		strings.NewReader(string(body)),
	)
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
		Code int                 `json:"code"`
		Data *oauthTokenResponse `json:"data"`
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

func (s *AuthService) fetchUserInfo(accessToken string) (*oauthUserInfo, error) {
	req, err := http.NewRequest("GET", s.oauthCfg.ServerURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("创建 userinfo 请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
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
		Code int            `json:"code"`
		Data *oauthUserInfo `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("解析 userinfo 响应失败: %w, body: %s", err, string(respBody))
	}
	if wrapper.Code != 0 || wrapper.Data == nil {
		return nil, fmt.Errorf("userinfo 返回错误: code=%d, body: %s", wrapper.Code, string(respBody))
	}
	return wrapper.Data, nil
}

func (s *AuthService) revokeToken(refreshToken string) error {
	payload, err := json.Marshal(map[string]string{"token": refreshToken})
	if err != nil {
		return fmt.Errorf("序列化 revoke 请求失败: %w", err)
	}
	resp, err := http.Post(
		s.oauthCfg.ServerURL+"/oauth/revoke",
		"application/json",
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RefreshOAuthToken refreshes the OAuth tokens using the refresh token.
func (s *AuthService) RefreshOAuthToken(refreshToken string) (*oauthTokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     s.oauthCfg.ClientID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化刷新请求失败: %w", err)
	}

	resp, err := http.Post(
		s.oauthCfg.ServerURL+"/oauth/token",
		"application/json",
		strings.NewReader(string(body)),
	)
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
		Code int                 `json:"code"`
		Data *oauthTokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}
	if wrapper.Code != 0 || wrapper.Data == nil || wrapper.Data.AccessToken == "" {
		return nil, fmt.Errorf("刷新 token 失败: %s", string(respBody))
	}
	return wrapper.Data, nil
}

// ──────────────────────────────────────────
// User find/create logic
// ──────────────────────────────────────────

func (s *AuthService) findOrCreateUser(oauthUser *oauthUserInfo) (*model.User, *errors.AppError) {
	// 1. Try to find by OAuth sub
	user, err := s.userRepo.FindByOAuthSub(oauthUser.Sub)
	if err == nil {
		return user, nil
	}

	// 2. Try to find by email (legacy user migration)
	user, err = s.userRepo.FindByEmail(oauthUser.Email)
	if err == nil {
		// Link the existing user to OAuth
		if linkErr := s.userRepo.LinkOAuthAccount(user.ID, oauthUser.Sub); linkErr != nil {
			return nil, errors.ErrInternal("关联 OAuth 账号失败")
		}
		return user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, errors.ErrInternal("查询用户失败")
	}

	// 3. Create new user
	newUser := &model.User{
		Name:        oauthUser.Name,
		Email:       oauthUser.Email,
		Password:    "",
		Avatar:      oauthUser.Picture,
		Role:        1,
		Moemoepoint: 7,
	}
	if err := s.userRepo.Create(newUser); err != nil {
		return nil, errors.ErrInternal("创建用户失败")
	}

	if err := s.userRepo.LinkOAuthAccount(newUser.ID, oauthUser.Sub); err != nil {
		return nil, errors.ErrInternal("关联 OAuth 账号失败")
	}

	return newUser, nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
