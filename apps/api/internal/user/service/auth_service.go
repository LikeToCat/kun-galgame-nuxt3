package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	mathrand "math/rand/v2"
	"time"

	"kun-galgame-api/internal/infrastructure/mail"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/model"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	rdb         *redis.Client
	oauthClient *oauth.Client
	mailer      *mail.Mailer
}

func NewAuthService(
	userRepo *repository.UserRepository,
	rdb *redis.Client,
	oauthClient *oauth.Client,
	mailer *mail.Mailer,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		rdb:         rdb,
		oauthClient: oauthClient,
		mailer:      mailer,
	}
}

// OAuthCallback exchanges the authorization code for tokens,
// fetches user info, finds or creates the local user, and creates a session.
func (s *AuthService) OAuthCallback(ctx context.Context, req *dto.OAuthCallbackRequest) (*dto.SessionResponse, *errors.AppError) {
	// 1. Exchange code for tokens
	// NOTE: /oauth/token returns raw OAuth format (NOT wrapped in {code, data})
	tokenResp, err := s.oauthClient.ExchangeCode(req.Code, req.CodeVerifier)
	if err != nil {
		return nil, errors.ErrBadRequest(fmt.Sprintf("OAuth 授权码交换失败: %v", err))
	}

	// 2. Fetch user info from OAuth server
	// NOTE: /oauth/userinfo returns wrapped format {code, data: {...}}
	oauthUser, err := s.oauthClient.FetchUserInfo(tokenResp.AccessToken)
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
			_ = s.oauthClient.RevokeToken(session.OAuthRefreshToken)
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
// User find/create logic
// ──────────────────────────────────────────

func (s *AuthService) findOrCreateUser(oauthUser *oauth.UserInfo) (*model.User, *errors.AppError) {
	// 1. Try to find by OAuth sub (already linked)
	user, err := s.userRepo.FindByOAuthSub(oauthUser.Sub)
	if err == nil {
		return user, nil
	}

	// 2. Try to find by email (legacy migrated user)
	if oauthUser.Email != "" {
		user, err = s.userRepo.FindByEmail(oauthUser.Email)
		if err == nil {
			if linkErr := s.userRepo.LinkOAuthAccount(user.ID, oauthUser.Sub); linkErr != nil {
				return nil, errors.ErrInternal("关联 OAuth 账号失败")
			}
			return user, nil
		}
	}

	// 3. Try to find by name (migrated user with same username)
	user, err = s.userRepo.FindByName(oauthUser.Name)
	if err == nil {
		if linkErr := s.userRepo.LinkOAuthAccount(user.ID, oauthUser.Sub); linkErr != nil {
			return nil, errors.ErrInternal("关联 OAuth 账号失败")
		}
		return user, nil
	}

	// 4. Create new user (deduplicate name if needed)
	name := oauthUser.Name
	for i := 1; ; i++ {
		exists, _ := s.userRepo.UsernameExists(name)
		if !exists {
			break
		}
		name = fmt.Sprintf("%s_%d", oauthUser.Name, i)
	}

	newUser := &model.User{
		Name:        name,
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

// SendResetEmailCode issues a 7-digit code for the "change email" flow.
// The code is stored in Redis under "{salt}:{email}" with 10-min TTL; rate
// limited per-email and per-IP for 30 seconds. Matches the legacy nitro
// shape: returns the salt, code is delivered by email.
//
// The caller must provide clientIP to enforce the per-IP rate limit.
func (s *AuthService) SendResetEmailCode(
	ctx context.Context,
	email, clientIP string,
) (string, *errors.AppError) {
	// The reset flow is for a user CHANGING their email. The target email
	// must not already belong to another account.
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return "", errors.ErrBadRequest("该邮箱已被注册")
	}

	// 30s rate limit on both email and IP
	if n, _ := s.rdb.Exists(ctx, "limit:email:"+email).Result(); n > 0 {
		return "", errors.ErrBadRequest("您的发邮件的频率过快, 请 30s 后重试")
	}
	if clientIP != "" {
		if n, _ := s.rdb.Exists(ctx, "limit:ip:"+clientIP).Result(); n > 0 {
			return "", errors.ErrBadRequest("您的发邮件的频率过快, 请 30s 后重试")
		}
	}

	code := randomDigitCode(7)
	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", errors.ErrInternal("生成 salt 失败")
	}
	salt := hex.EncodeToString(saltBytes)

	s.rdb.Set(ctx, salt+":"+email, code, 10*time.Minute)
	s.rdb.Set(ctx, "limit:email:"+email, "1", 30*time.Second)
	if clientIP != "" {
		s.rdb.Set(ctx, "limit:ip:"+clientIP, "1", 30*time.Second)
	}

	// Send the email in the background — don't block on SMTP; log failures.
	if s.mailer != nil {
		go func(to, c string) {
			body := fmt.Sprintf(
				"Your verification code is\n%s\nYou are resetting your email. Please do not disclose this verification code.",
				c,
			)
			if err := s.mailer.Send(to, "KUN Visual Novel Verification Code", body); err != nil {
				slog.Warn("发送验证邮件失败", "email", to, "error", err)
			}
		}(email, code)
	} else {
		slog.Warn("邮件服务未配置，跳过发送验证码", "email", email, "code", code)
	}

	return salt, nil
}

// randomDigitCode returns a string of n decimal digits.
func randomDigitCode(n int) string {
	digits := make([]byte, n)
	for i := range digits {
		digits[i] = byte('0' + mathrand.IntN(10))
	}
	return string(digits)
}
