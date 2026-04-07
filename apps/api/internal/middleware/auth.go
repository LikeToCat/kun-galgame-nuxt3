package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const UserInfoKey contextKey = "userInfo"

// UserInfo represents the authenticated user extracted from session.
type UserInfo struct {
	UID   int    `json:"uid"`
	Sub   string `json:"sub"` // OAuth UUID
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  int    `json:"role"`
}

// SessionData is stored in Redis under "session:{token}".
type SessionData struct {
	UserInfo
	OAuthAccessToken  string `json:"oauth_access_token"`
	OAuthRefreshToken string `json:"oauth_refresh_token"`
	OAuthExpiresAt    int64  `json:"oauth_expires_at"`
}

// Auth creates a middleware that validates the session cookie.
// It looks up the session in Redis and attaches UserInfo to the context.
func Auth(rdb *redis.Client, oauthCfg config.OAuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("kun_session")
		if token == "" {
			return response.Error(c, errors.ErrAuthExpired())
		}

		ctx := context.Background()
		val, err := rdb.Get(ctx, "session:"+token).Result()
		if err != nil {
			return response.Error(c, errors.ErrAuthExpired())
		}

		var session SessionData
		if err := json.Unmarshal([]byte(val), &session); err != nil {
			return response.Error(c, errors.ErrAuthExpired())
		}

		// If OAuth access token is expired, try to refresh it
		if session.OAuthExpiresAt > 0 && time.Now().Unix() > session.OAuthExpiresAt {
			refreshed, refreshErr := refreshOAuthToken(oauthCfg, session.OAuthRefreshToken)
			if refreshErr != nil {
				rdb.Del(ctx, "session:"+token)
				return response.Error(c, errors.ErrAuthExpired())
			}
			session.OAuthAccessToken = refreshed.AccessToken
			session.OAuthRefreshToken = refreshed.RefreshToken
			session.OAuthExpiresAt = time.Now().Unix() + int64(refreshed.ExpiresIn)

			data, _ := json.Marshal(session)
			rdb.Set(ctx, "session:"+token, data, 7*24*time.Hour)
		}

		c.Locals(string(UserInfoKey), &session.UserInfo)
		return c.Next()
	}
}

// OptionalAuth is like Auth but does not fail if no session is present.
// If a valid session exists, UserInfo is attached; otherwise the request proceeds.
func OptionalAuth(rdb *redis.Client, oauthCfg config.OAuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("kun_session")
		if token == "" {
			return c.Next()
		}

		ctx := context.Background()
		val, err := rdb.Get(ctx, "session:"+token).Result()
		if err != nil {
			return c.Next()
		}

		var session SessionData
		if err := json.Unmarshal([]byte(val), &session); err != nil {
			return c.Next()
		}

		c.Locals(string(UserInfoKey), &session.UserInfo)
		return c.Next()
	}
}

// GetUser extracts UserInfo from the Fiber context. Returns nil if not authenticated.
func GetUser(c *fiber.Ctx) *UserInfo {
	info, ok := c.Locals(string(UserInfoKey)).(*UserInfo)
	if !ok {
		return nil
	}
	return info
}

// MustGetUser extracts UserInfo or returns an auth error.
func MustGetUser(c *fiber.Ctx) (*UserInfo, *errors.AppError) {
	info := GetUser(c)
	if info == nil {
		return nil, errors.ErrAuthExpired()
	}
	return info, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func refreshOAuthToken(cfg config.OAuthConfig, refreshToken string) (*tokenResponse, error) {
	body := fmt.Sprintf(
		`{"grant_type":"refresh_token","refresh_token":"%s","client_id":"%s"}`,
		refreshToken, cfg.ClientID,
	)

	resp, err := http.Post(
		cfg.ServerURL+"/oauth/token",
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var wrapper struct {
		Code int            `json:"code"`
		Data *tokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, err
	}
	if wrapper.Code != 0 || wrapper.Data == nil {
		return nil, fmt.Errorf("OAuth token 刷新失败")
	}
	return wrapper.Data, nil
}
