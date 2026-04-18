package handler

import (
	"log/slog"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type OAuthHandler struct {
	authService *service.AuthService
	secure      bool // true in production (HTTPS), false in dev (HTTP)
}

func NewOAuthHandler(authService *service.AuthService, isProd bool) *OAuthHandler {
	return &OAuthHandler{authService: authService, secure: isProd}
}

// Callback handles the OAuth code exchange: code → token → userinfo → session.
// POST /api/auth/oauth/callback
func (h *OAuthHandler) Callback(c *fiber.Ctx) error {
	var req dto.OAuthCallbackRequest
	slog.Debug("OAuth callback",
		"content-type", c.Get("Content-Type"),
		"body", string(c.Body()),
	)

	if err := utils.ParseAndValidate(c, &req); err != nil {
		slog.Error("OAuth callback 验证失败", "error", err.Message, "body", string(c.Body()))
		return response.Error(c, err)
	}

	session, appErr := h.authService.OAuthCallback(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "kun_session",
		Value:    session.Token,
		MaxAge:   7 * 24 * 3600, // 7 days
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OK(c, session.User)
}

// Logout clears the session.
// POST /api/auth/logout
func (h *OAuthHandler) Logout(c *fiber.Ctx) error {
	token := c.Cookies("kun_session")
	if token != "" {
		_ = h.authService.Logout(c.Context(), token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "kun_session",
		Value:    "",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OKMessage(c, "已登出")
}

// SendResetEmailCode issues a 7-digit verification code for the change-email
// flow. The email must not already belong to any user. Returns the opaque
// salt; the code is delivered via email.
// POST /api/auth/email/code/reset
func (h *OAuthHandler) SendResetEmailCode(c *fiber.Ctx) error {
	var req dto.SendResetCodeRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	salt, appErr := h.authService.SendResetEmailCode(c.Context(), req.Email, c.IP())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, salt)
}

// Me returns the current authenticated user's profile.
// GET /api/auth/me
func (h *OAuthHandler) Me(c *fiber.Ctx) error {
	user, err := middleware.MustGetUser(c)
	if err != nil {
		return response.Error(c, err)
	}

	profile, appErr := h.authService.GetProfile(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, profile)
}
