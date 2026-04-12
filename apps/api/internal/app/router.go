package app

import (
	"time"

	"kun-galgame-api/internal/middleware"

	fiberCors "github.com/gofiber/fiber/v2/middleware/cors"
)

func (a *App) setupRoutes() {
	a.Fiber.Use(fiberCors.New(middleware.CORS(a.Config.CORS.AllowOrigins)))

	api := a.Fiber.Group("/api")

	// ── Public routes ──────────────────────────
	api.Get("/home", a.HomeHandler.GetHome)

	// ── Auth routes (public) ───────────────────
	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)

	// ── Auth routes (authenticated) ────────────
	authed := api.Group("", middleware.Auth(a.Redis, a.Config.OAuth))
	authed.Get("/auth/me", a.OAuthHandler.Me)

	// Rate limiters for sensitive mutations
	checkInRL := middleware.RateLimit(a.Redis, "checkin", 1, 24*time.Hour)
	usernameRL := middleware.RateLimit(a.Redis, "username", 3, time.Hour)
	emailRL := middleware.RateLimit(a.Redis, "email", 3, time.Hour)
	avatarRL := middleware.RateLimit(a.Redis, "avatar", 5, time.Hour)

	// ── User routes (authenticated, fixed paths — must be before :uid) ──
	authed.Post("/user/check-in", checkInRL, a.UserHandler.CheckIn)
	authed.Put("/user/bio", a.UserHandler.UpdateBio)
	authed.Put("/user/username", usernameRL, a.UserHandler.UpdateUsername)
	authed.Put("/user/email", emailRL, a.UserHandler.UpdateEmail)
	authed.Get("/user/email", a.UserHandler.GetEmail)
	authed.Get("/user/status", a.UserHandler.GetStatus)
	authed.Post("/user/avatar", avatarRL, a.UserHandler.UploadAvatar)

	// ── User routes (public, parameterized — after fixed paths) ─────
	api.Get("/user/:uid", a.UserHandler.GetProfile)
	api.Get("/user/:uid/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:uid/topics", a.UserHandler.GetUserTopics)

	// ── User admin routes ──────────────────────
	admin := authed.Group("", middleware.RequireRole(3))
	admin.Put("/user/:uid/ban", a.UserHandler.BanUser)
	admin.Delete("/user/:uid", a.UserHandler.DeleteUser)
}
