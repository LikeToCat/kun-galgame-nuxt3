package middleware

import (
	"encoding/json"
	"log/slog"
	"time"

	"kun-galgame-api/internal/user/oauth"
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
//
// Take an *oauth.Client (the same one AuthService uses) so that token
// refresh logic lives in exactly one place — see oauth.Client.RefreshOAuthToken.
func Auth(rdb *redis.Client, oauthClient *oauth.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("kun_session")
		if token == "" {
			return response.Error(c, errors.ErrAuthExpired())
		}

		ctx := c.Context()
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
			// Use Redis SETNX as a distributed lock to prevent concurrent refreshes
			lockKey := "refresh_lock:" + token
			locked, _ := rdb.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
			if locked {
				defer rdb.Del(ctx, lockKey)
				refreshed, refreshErr := oauthClient.RefreshOAuthToken(session.OAuthRefreshToken)
				if refreshErr != nil {
					slog.Warn("OAuth token 刷新失败", "error", refreshErr)
					rdb.Del(ctx, "session:"+token)
					return response.Error(c, errors.ErrAuthExpired())
				}
				session.OAuthAccessToken = refreshed.AccessToken
				session.OAuthRefreshToken = refreshed.RefreshToken
				session.OAuthExpiresAt = time.Now().Unix() + int64(refreshed.ExpiresIn)

				data, err := json.Marshal(session)
				if err != nil {
					slog.Error("序列化 session 失败", "error", err)
					return response.Error(c, errors.ErrInternal("服务器内部错误"))
				}
				rdb.Set(ctx, "session:"+token, data, 7*24*time.Hour)
			} else {
				// Another request is refreshing, re-read session from Redis
				val, err = rdb.Get(ctx, "session:"+token).Result()
				if err != nil {
					return response.Error(c, errors.ErrAuthExpired())
				}
				if err := json.Unmarshal([]byte(val), &session); err != nil {
					return response.Error(c, errors.ErrAuthExpired())
				}
			}
		}

		c.Locals(string(UserInfoKey), &session.UserInfo)
		return c.Next()
	}
}

// OptionalAuth is like Auth but does not fail if no session is present.
// If a valid session exists, UserInfo is attached; otherwise the request proceeds.
func OptionalAuth(rdb *redis.Client, _ *oauth.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("kun_session")
		if token == "" {
			return c.Next()
		}

		ctx := c.Context()
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

