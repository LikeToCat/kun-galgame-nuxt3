package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

// AuthService handles the OAuth-callback session bootstrap and a few
// session-lifecycle ops. Identity (name / avatar / email / bio / status)
// is owned by OAuth — kungal only persists per-site state in
// kungal_user_state (moemoepoint, daily counters).
type AuthService struct {
	stateRepo   *repository.StateRepository
	rdb         *redis.Client
	oauthClient *oauth.Client
	userClient  *userclient.Client
}

func NewAuthService(
	stateRepo *repository.StateRepository,
	rdb *redis.Client,
	oauthClient *oauth.Client,
	userClient *userclient.Client,
) *AuthService {
	return &AuthService{
		stateRepo:   stateRepo,
		rdb:         rdb,
		oauthClient: oauthClient,
		userClient:  userClient,
	}
}

// OAuthCallback exchanges the authorization code for tokens, fetches user
// info from OAuth (which is the single source of truth for identity), and
// creates a kungal session. Idempotently ensures kungal_user_state(uid)
// exists so the new user starts with the default 7 moemoepoint balance.
func (s *AuthService) OAuthCallback(
	ctx context.Context,
	req *dto.OAuthCallbackRequest,
) (*dto.SessionResponse, *errors.AppError) {
	tokenResp, err := s.oauthClient.ExchangeCode(req.Code, req.CodeVerifier)
	if err != nil {
		return nil, errors.ErrBadRequest(fmt.Sprintf("OAuth 授权码交换失败: %v", err))
	}

	oauthUser, err := s.oauthClient.FetchUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, errors.ErrBadRequest(fmt.Sprintf("获取 OAuth 用户信息失败: %v", err))
	}
	if oauthUser.ID <= 0 {
		// Hard-fail if the OAuth server hasn't yet been updated to include
		// the integer `id` field on /oauth/userinfo. See the comment on
		// oauth.UserInfo for the rationale.
		return nil, errors.ErrInternal(
			"OAuth /oauth/userinfo 未返回用户 id; 请确认 OAuth server 已更新",
		)
	}

	// First-time login on this site: create the kungal-state row. No-op
	// for returning users.
	if err := s.stateRepo.Ensure(oauthUser.ID); err != nil {
		return nil, errors.ErrInternal("初始化用户状态失败")
	}

	// Pull moemoepoint for the SessionResponse so the frontend can show it
	// immediately without a follow-up /user/status call.
	state, _ := s.stateRepo.FindByID(oauthUser.ID)
	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}

	role := middleware.RoleFromOAuthRoles(oauthUser.Roles)

	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, errors.ErrInternal("生成会话令牌失败")
	}

	sessionData := middleware.SessionData{
		UserInfo: middleware.UserInfo{
			UID:   oauthUser.ID,
			Sub:   oauthUser.Sub,
			Name:  oauthUser.Name,
			Email: oauthUser.Email,
			Role:  role,
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
			ID:          oauthUser.ID,
			Name:        oauthUser.Name,
			Email:       oauthUser.Email,
			Avatar:      oauthUser.Picture,
			Role:        role,
			Moemoepoint: moe,
			Bio:         "", // bio is OAuth-owned, available via /auth/me
		},
	}, nil
}

// Logout deletes the session from Redis and revokes the OAuth token.
func (s *AuthService) Logout(ctx context.Context, sessionToken string) error {
	val, err := s.rdb.Get(ctx, "session:"+sessionToken).Result()
	if err == nil {
		var session middleware.SessionData
		if json.Unmarshal([]byte(val), &session) == nil && session.OAuthRefreshToken != "" {
			_ = s.oauthClient.RevokeToken(session.OAuthRefreshToken)
		}
	}
	return s.rdb.Del(ctx, "session:"+sessionToken).Err()
}

// GetProfile returns the full profile for the currently logged-in user.
// Identity fields come from OAuth (via userclient); moemoepoint and other
// per-site state come from kungal_user_state. Used by GET /api/auth/me.
func (s *AuthService) GetProfile(
	ctx context.Context,
	userID int,
) (*dto.UserProfile, *errors.AppError) {
	u, ok, err := s.userClient.User(ctx, userID)
	if err != nil {
		return nil, errors.ErrInternal("查询用户信息失败")
	}
	if !ok {
		return nil, errors.ErrNotFound("用户不存在")
	}
	state, _ := s.stateRepo.FindByID(userID)
	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}
	return &dto.UserProfile{
		ID:          u.ID,
		Name:        u.Name,
		Email:       "", // not in /users/batch (privacy); fetch via /oauth/userinfo client-side
		Avatar:      u.Avatar,
		Role:        middleware.RoleFromOAuthRoles(u.Roles),
		Moemoepoint: moe,
		Bio:         u.Bio,
	}, nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
