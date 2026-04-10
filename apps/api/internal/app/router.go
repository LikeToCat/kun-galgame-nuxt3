package app

import (
	"kun-galgame-api/internal/middleware"

	fiberCors "github.com/gofiber/fiber/v2/middleware/cors"
)

func (a *App) setupRoutes() {
	a.Fiber.Use(fiberCors.New(middleware.CORS("http://127.0.0.1:2333,https://www.kungal.com")))

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

	// ── User routes (public) ───────────────────
	api.Get("/user/:uid", a.UserHandler.GetProfile)
	api.Get("/user/:uid/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:uid/topics", a.UserHandler.GetUserTopics)

	// ── User routes (authenticated) ────────────
	authed.Post("/user/check-in", a.UserHandler.CheckIn)
	authed.Put("/user/bio", a.UserHandler.UpdateBio)
	authed.Put("/user/username", a.UserHandler.UpdateUsername)
	authed.Put("/user/email", a.UserHandler.UpdateEmail)
	authed.Get("/user/email", a.UserHandler.GetEmail)
	authed.Get("/user/status", a.UserHandler.GetStatus)
	authed.Post("/user/avatar", a.UserHandler.UploadAvatar)

	// ── User admin routes ──────────────────────
	admin := authed.Group("", middleware.RequireRole(3))
	admin.Put("/user/:uid/ban", a.UserHandler.BanUser)
	admin.Delete("/user/:uid", a.UserHandler.DeleteUser)
}
