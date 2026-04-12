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

	// ── Topic routes (public, optional auth) ──
	optAuth := api.Group("", middleware.OptionalAuth(a.Redis, a.Config.OAuth))
	optAuth.Get("/topic", a.TopicHandler.GetList)
	optAuth.Get("/topic/:tid", a.TopicHandler.GetDetail)
	optAuth.Get("/topic/:tid/reply", a.ReplyHandler.GetReplies)
	optAuth.Get("/topic/:tid/reply/detail", a.ReplyHandler.GetReplyDetail)
	optAuth.Get("/topic/:tid/poll/topic", a.PollHandler.GetPollsByTopic)
	optAuth.Get("/topic/:tid/poll/log", a.PollHandler.GetVoteLog)

	// ── Topic routes (authenticated) ──
	authed.Post("/topic", a.TopicHandler.Create)
	authed.Put("/topic/:tid", a.TopicHandler.Update)
	authed.Put("/topic/:tid/like", a.TopicHandler.ToggleLike)
	authed.Put("/topic/:tid/dislike", a.TopicHandler.ToggleDislike)
	authed.Put("/topic/:tid/upvote", a.TopicHandler.Upvote)
	authed.Put("/topic/:tid/favorite", a.TopicHandler.ToggleFavorite)
	authed.Put("/topic/:tid/hide", a.TopicHandler.ToggleHide)
	authed.Put("/topic/:tid/best-answer", a.TopicHandler.SetBestAnswer)

	// ── Reply routes (authenticated) ──
	authed.Post("/topic/:tid/reply", a.ReplyHandler.CreateReply)
	authed.Put("/topic/:tid/reply", a.ReplyHandler.UpdateReply)
	authed.Delete("/topic/:tid/reply", a.ReplyHandler.DeleteReply)
	authed.Put("/topic/:tid/reply/like", a.ReplyHandler.ToggleReplyLike)
	authed.Put("/topic/:tid/reply/dislike", a.ReplyHandler.ToggleReplyDislike)
	authed.Put("/topic/:tid/reply/pin", a.ReplyHandler.PinReply)

	// ── Comment routes (authenticated) ──
	authed.Post("/topic/:tid/comment", a.ReplyHandler.CreateComment)
	authed.Put("/topic/:tid/comment/like", a.ReplyHandler.ToggleCommentLike)
	authed.Delete("/topic/:tid/comment", a.ReplyHandler.DeleteComment)

	// ── Poll routes (authenticated) ──
	authed.Post("/topic/:tid/poll", a.PollHandler.CreatePoll)
	authed.Delete("/topic/:tid/poll", a.PollHandler.DeletePoll)
	authed.Post("/topic/:tid/poll/vote", a.PollHandler.Vote)

	// ── Message routes (authenticated) ──
	authed.Get("/message", a.MessageHandler.GetMessages)
	authed.Delete("/message/:id", a.MessageHandler.DeleteMessage)
	authed.Put("/message/system/read", a.MessageHandler.MarkAllRead)
	authed.Put("/message/admin/read", a.MessageHandler.MarkAdminRead)
	authed.Get("/message/nav/system", a.MessageHandler.GetNavSummary)

	// ── Message routes (public) ──
	api.Get("/message/admin", a.MessageHandler.GetSystemMessages)

	// ── Admin routes (role >= 3) ──
	admin.Get("/admin/overview/all", a.AdminHandler.GetOverview)
	admin.Get("/admin/overview/stats", a.AdminHandler.GetStats)
	admin.Put("/admin/setting/register", a.AdminHandler.ToggleRegisterSetting)
	adminRead := authed.Group("", middleware.RequireRole(2))
	adminRead.Get("/admin/user", a.AdminHandler.GetUserList)
	adminRead.Get("/admin/user/search", a.AdminHandler.SearchUsers)

	// ── Admin setting (public read) ──
	api.Get("/admin/setting/register", a.AdminHandler.GetRegisterSetting)

	// ── Ranking routes (public) ──
	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	// ── Section & Category routes (public) ──
	api.Get("/section", a.SectionHandler.GetSectionTopics)
	api.Get("/category", a.SectionHandler.GetCategories)
}
