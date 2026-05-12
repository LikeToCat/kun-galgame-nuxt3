package middleware

import (
	"context"
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

const (
	UserInfoKey         contextKey = "userInfo"
	OAuthAccessTokenKey contextKey = "oauthAccessToken"
)

// UserInfo represents the authenticated user extracted from session.
type UserInfo struct {
	UID   int    `json:"uid"`
	Sub   string `json:"sub"` // OAuth UUID
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  int    `json:"role"` // derived from OAuth roles claim — see RoleFromOAuthRoles
}

// RoleFromOAuthRoles maps the OAuth `roles` claim (e.g. ["user", "admin"])
// to kungal's internal numeric hierarchy used by RequireRole(N) middleware:
//
//	3 — super admin (delete users, sensitive admin ops)
//	2 — moderator  (ban, content moderation, doc/website/update writes)
//	1 — normal user
//
// Anything else falls through to 1. The mapping is centralized here so
// every place that derives Role from OAuth uses the same rules.
func RoleFromOAuthRoles(roles []string) int {
	for _, r := range roles {
		if r == "admin" || r == "super_admin" {
			return 3
		}
	}
	for _, r := range roles {
		if r == "moderator" || r == "mod" {
			return 2
		}
	}
	return 1
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
// Identity (name / avatar / etc.) is OAuth-owned post-migration; mappers
// fetch via pkg/userclient.
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

		// Refresh the OAuth access token if it's expired (or within a 30s
		// grace window of expiry — see refreshSkew below). This is the hot
		// path that runs on every authenticated request, so the logic
		// needs to:
		//   (a) handle concurrent expiry across multiple in-flight requests
		//       without doing N parallel refresh round-trips, and
		//   (b) survive transient OAuth failures without killing the
		//       session and kicking the user out for an OAuth hiccup.
		//
		// Strategy: SETNX-based single-flight lock. The winner does the
		// refresh; losers wait (poll) until the winner publishes the new
		// session, then proceed with the fresh tokens. On refresh failure
		// we return 205 to THIS request but leave the session intact so
		// the next request can retry — only a permanently-invalid refresh
		// token will keep failing, and we'd rather get many 205s during a
		// transient outage than auto-logout every active user.
		const refreshSkew = 30 * time.Second
		needsRefresh := session.OAuthExpiresAt > 0 &&
			time.Now().Add(refreshSkew).Unix() > session.OAuthExpiresAt
		if needsRefresh {
			lockKey := "refresh_lock:" + token
			// Lock TTL must exceed the OAuth client's HTTP timeout (10s) so
			// the lock isn't auto-released mid-refresh — otherwise a second
			// request would grab it and call OAuth with a refresh token
			// that's already been rotated by the first.
			locked, _ := rdb.SetNX(ctx, lockKey, "1", 15*time.Second).Result()
			if locked {
				if err := refreshSession(ctx, rdb, oauthClient, token, &session); err != nil {
					rdb.Del(ctx, lockKey)
					return response.Error(c, err)
				}
				rdb.Del(ctx, lockKey)
			} else {
				if err := waitForRefresh(ctx, rdb, lockKey, token, &session); err != nil {
					return response.Error(c, err)
				}
			}
		}

		c.Locals(string(UserInfoKey), &session.UserInfo)
		// Expose the session's OAuth access token to handlers that need to
		// forward authority to the wiki service. Sourcing this from Redis
		// (rather than a client-supplied X-OAuth-Token header) guarantees
		// the token's subject matches the kun_session cookie holder.
		c.Locals(string(OAuthAccessTokenKey), session.OAuthAccessToken)
		return c.Next()
	}
}

// OptionalAuth is like Auth but does not fail if no session is present.
// If a valid session exists, UserInfo is attached; otherwise the request proceeds.
func OptionalAuth(rdb *redis.Client) fiber.Handler {
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

// GetAccessToken returns the session-stored OAuth access token attached by
// Auth middleware. Returns empty when no valid session is present (e.g.
// OptionalAuth path with anonymous user). Callers that forward authority to
// the wiki service MUST source the token from here — never from a
// client-supplied header — so the token subject is guaranteed to match the
// kun_session cookie holder.
func GetAccessToken(c *fiber.Ctx) string {
	tok, _ := c.Locals(string(OAuthAccessTokenKey)).(string)
	return tok
}

// refreshSession is the lock-winner path: actually call OAuth, mutate the
// passed-in session in place, and write the result back to Redis.
//
// Failure branches (matters for the user-experience side of the 401 loop):
//   - oauth.IsBanned(err)            → delete session, surface CodeBanned;
//                                      frontend stops the user from looping
//                                      through /login (a re-login hits 10014
//                                      again at the very next refresh).
//   - oauth.IsRefreshTokenDead(err)  → delete session, surface 205; user
//                                      must do a fresh /oauth/authorize.
//                                      Covers refresh_token expired, client_id
//                                      mismatch, invalid_grant, secret mismatch.
//   - oauth.IsTransient(err)         → keep session, surface 205; the next
//                                      request retries the refresh. This is
//                                      what makes OAuth restarts / network
//                                      hiccups not auto-logout every user.
func refreshSession(
	ctx context.Context,
	rdb *redis.Client,
	oauthClient *oauth.Client,
	token string,
	session *SessionData,
) *errors.AppError {
	refreshed, err := oauthClient.RefreshOAuthToken(session.OAuthRefreshToken)
	if err != nil {
		switch {
		case oauth.IsBanned(err):
			slog.Warn("OAuth 刷新返回账号封禁", "error", err)
			rdb.Del(ctx, "session:"+token)
			return errors.ErrAccountBanned()
		case oauth.IsRefreshTokenDead(err):
			slog.Warn("OAuth refresh_token 不可恢复, 清除 session", "error", err)
			rdb.Del(ctx, "session:"+token)
			return errors.ErrAuthExpired()
		default:
			// Transient: don't touch the session, let the next request retry.
			slog.Warn("OAuth token 刷新失败 (保留 session, 留给下次请求重试)",
				"error", err)
			return errors.ErrAuthExpired()
		}
	}
	session.OAuthAccessToken = refreshed.AccessToken
	session.OAuthRefreshToken = refreshed.RefreshToken
	session.OAuthExpiresAt = time.Now().Unix() + int64(refreshed.ExpiresIn)

	data, mErr := json.Marshal(session)
	if mErr != nil {
		slog.Error("序列化 session 失败", "error", mErr)
		return errors.ErrInternal("服务器内部错误")
	}
	rdb.Set(ctx, "session:"+token, data, 7*24*time.Hour)
	return nil
}

// waitForRefresh is the lock-loser path. Another request is currently
// refreshing this user's token; instead of racing through with the stale
// access token (which would just generate downstream 401s from the wiki
// service), we poll until either:
//
//   - the session in Redis has a fresh OAuthExpiresAt → proceed with the
//     freshly-published tokens; or
//   - the lock key disappears with the session still expired → the winner
//     failed; surface as auth-expired so the next request can retry; or
//   - we exceed the wait deadline → also surface as auth-expired.
//
// The poll interval (150ms) gives sub-second responsiveness once the
// winner publishes. The deadline (12s) sits between the OAuth client's
// 10s HTTP timeout and the 15s SETNX TTL, so a still-pending refresh has
// time to finish but we give up before the lock would auto-expire (after
// which we wouldn't be able to distinguish "refresh failed" from
// "refresh still in flight").
func waitForRefresh(
	ctx context.Context,
	rdb *redis.Client,
	lockKey, token string,
	session *SessionData,
) *errors.AppError {
	deadline := time.Now().Add(12 * time.Second)
	for {
		time.Sleep(150 * time.Millisecond)

		val, err := rdb.Get(ctx, "session:"+token).Result()
		if err != nil {
			return errors.ErrAuthExpired()
		}
		if uErr := json.Unmarshal([]byte(val), session); uErr != nil {
			return errors.ErrAuthExpired()
		}

		// Refresh published — fall through to the request handler.
		if session.OAuthExpiresAt > time.Now().Unix() {
			return nil
		}

		// Lock released but session still expired → winner's refresh failed.
		// Fail fast (don't wait full deadline) so the user just retries.
		exists, _ := rdb.Exists(ctx, lockKey).Result()
		if exists == 0 {
			return errors.ErrAuthExpired()
		}

		if time.Now().After(deadline) {
			return errors.ErrAuthExpired()
		}
	}
}


