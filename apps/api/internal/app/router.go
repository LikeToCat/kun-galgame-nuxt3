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

	// ���─ Auth routes (authenticated) ────────────
	authed := api.Group("", middleware.Auth(a.Redis, a.Config.OAuth))
	authed.Get("/auth/me", a.OAuthHandler.Me)

	// Rate limiters
	checkInRL := middleware.RateLimit(a.Redis, "checkin", 1, 24*time.Hour)
	usernameRL := middleware.RateLimit(a.Redis, "username", 3, time.Hour)
	emailRL := middleware.RateLimit(a.Redis, "email", 3, time.Hour)
	avatarRL := middleware.RateLimit(a.Redis, "avatar", 5, time.Hour)

	// ── User routes (authenticated, fixed paths) ──
	authed.Post("/user/check-in", checkInRL, a.UserHandler.CheckIn)
	authed.Put("/user/bio", a.UserHandler.UpdateBio)
	authed.Put("/user/username", usernameRL, a.UserHandler.UpdateUsername)
	authed.Put("/user/email", emailRL, a.UserHandler.UpdateEmail)
	authed.Get("/user/email", a.UserHandler.GetEmail)
	authed.Get("/user/status", a.UserHandler.GetStatus)
	authed.Post("/user/avatar", avatarRL, a.UserHandler.UploadAvatar)

	// ── User routes (public, parameterized) ──
	api.Get("/user/:uid", a.UserHandler.GetProfile)
	api.Get("/user/:uid/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:uid/topics", a.UserHandler.GetUserTopics)

	// ── User admin routes ──
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

	// ── Message routes ──
	authed.Get("/message", a.MessageHandler.GetMessages)
	authed.Delete("/message/:id", a.MessageHandler.DeleteMessage)
	authed.Put("/message/system/read", a.MessageHandler.MarkAllRead)
	authed.Put("/message/admin/read", a.MessageHandler.MarkAdminRead)
	authed.Get("/message/nav/system", a.MessageHandler.GetNavSummary)
	api.Get("/message/admin", a.MessageHandler.GetSystemMessages)

	// ── Admin routes ──
	admin.Get("/admin/overview/all", a.AdminHandler.GetOverview)
	admin.Get("/admin/overview/stats", a.AdminHandler.GetStats)
	admin.Put("/admin/setting/register", a.AdminHandler.ToggleRegisterSetting)
	api.Get("/admin/setting/register", a.AdminHandler.GetRegisterSetting)
	adminRead := authed.Group("", middleware.RequireRole(2))
	adminRead.Get("/admin/user", a.AdminHandler.GetUserList)
	adminRead.Get("/admin/user/search", a.AdminHandler.SearchUsers)

	// ── Ranking routes (public) ──
	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	// ── Section & Category (public) ���─
	api.Get("/section", a.SectionHandler.GetSectionTopics)
	api.Get("/category", a.SectionHandler.GetCategories)

	// ── Doc routes ──
	api.Get("/doc/article", a.DocHandler.GetArticles)
	api.Get("/doc/article/:slug", a.DocHandler.GetArticleBySlug)
	api.Get("/doc/category", a.DocHandler.GetCategories)
	api.Get("/doc/tag", a.DocHandler.GetTags)
	docAdmin := authed.Group("", middleware.RequireRole(2))
	docAdmin.Post("/doc/article", a.DocHandler.CreateArticle)
	docAdmin.Put("/doc/article", a.DocHandler.UpdateArticle)
	docAdmin.Delete("/doc/article", a.DocHandler.DeleteArticle)
	docAdmin.Post("/doc/category", a.DocHandler.CreateCategory)
	docAdmin.Put("/doc/category", a.DocHandler.UpdateCategory)
	docAdmin.Delete("/doc/category", a.DocHandler.DeleteCategory)
	docAdmin.Post("/doc/tag", a.DocHandler.CreateTag)
	docAdmin.Delete("/doc/tag", a.DocHandler.DeleteTag)

	// ── Website routes ──
	optAuth.Get("/website", a.WebsiteHandler.GetWebsites)
	optAuth.Get("/website/:domain", a.WebsiteHandler.GetWebsiteDetail)
	api.Get("/website-category/:name", a.WebsiteHandler.GetWebsiteCategory)
	api.Get("/website-tag", a.WebsiteHandler.GetWebsiteTags)
	wsAdmin := authed.Group("", middleware.RequireRole(2))
	wsAdmin.Post("/website", a.WebsiteHandler.CreateWebsite)
	wsAdmin.Put("/website/:domain", a.WebsiteHandler.UpdateWebsite)
	wsAdmin.Delete("/website/:domain", a.WebsiteHandler.DeleteWebsite)
	wsAdmin.Put("/website-category", a.WebsiteHandler.UpdateWebsiteCategory)
	wsAdmin.Post("/website-tag", a.WebsiteHandler.CreateWebsiteTag)
	wsAdmin.Put("/website-tag", a.WebsiteHandler.UpdateWebsiteTag)
	wsAdmin.Delete("/website-tag", a.WebsiteHandler.DeleteWebsiteTag)
	authed.Put("/website/:domain/like", a.WebsiteHandler.ToggleLike)
	authed.Put("/website/:domain/favorite", a.WebsiteHandler.ToggleFavorite)
	authed.Post("/website/:domain/comment", a.WebsiteHandler.CreateComment)
	authed.Delete("/website/:domain/comment", a.WebsiteHandler.DeleteComment)

	// ── Update routes ──
	api.Get("/update/history", a.UpdateHandler.GetHistory)
	api.Get("/update/todo", a.UpdateHandler.GetTodos)
	updateAdmin := authed.Group("", middleware.RequireRole(2))
	updateAdmin.Post("/update/history", a.UpdateHandler.CreateHistory)
	updateAdmin.Delete("/update/history", a.UpdateHandler.DeleteHistory)
	updateAdmin.Post("/update/todo", a.UpdateHandler.CreateTodo)
	updateAdmin.Delete("/update/todo", a.UpdateHandler.DeleteTodo)

	// ── Misc routes ──
	authed.Post("/report/submit", a.MiscHandler.SubmitReport)
	api.Get("/rss/topic", a.MiscHandler.GetTopicRSS)

	// ── Activity routes (public) ──
	api.Get("/activity", a.ActivityHandler.GetActivity)
	api.Get("/activity/timeline", a.ActivityHandler.GetTimeline)

	// ── Search route (public) ──
	api.Get("/search", a.SearchHandler.Search)

	// ── Image upload (authenticated) ──
	authed.Post("/image/topic", a.ImageHandler.UploadTopicImage)

	// ── Galgame routes ──
	// Aggregation (wiki metadata + local interaction data)
	optAuth.Get("/galgame/:gid", a.GalgameHandler.GetDetail)

	// Proxy to wiki service (with local side effects)
	authed.Post("/galgame", a.GalgameHandler.Create)
	authed.Put("/galgame/:gid/prs/:id/merge", a.GalgameHandler.MergePR)

	// Proxy to wiki service (no local side effects, pure forwarding)
	api.Get("/galgame", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/check", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev/diff", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/prs", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/prs/:id", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/links", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/aliases", a.GalgameHandler.ProxyGet)
	api.Get("/galgame/:gid/contributors", a.GalgameHandler.ProxyGet)
	api.Get("/tag", a.GalgameHandler.ProxyGet)
	api.Get("/tag/search", a.GalgameHandler.ProxyGet)
	api.Get("/tag/multi", a.GalgameHandler.ProxyGet)
	api.Get("/tag/:name", a.GalgameHandler.ProxyGet)
	api.Get("/official", a.GalgameHandler.ProxyGet)
	api.Get("/official/search", a.GalgameHandler.ProxyGet)
	api.Get("/official/:name", a.GalgameHandler.ProxyGet)
	api.Get("/engine", a.GalgameHandler.ProxyGet)
	api.Get("/engine/:name", a.GalgameHandler.ProxyGet)
	api.Get("/series", a.GalgameHandler.ProxyGet)
	api.Get("/series/search", a.GalgameHandler.ProxyGet)
	api.Get("/series/:id", a.GalgameHandler.ProxyGet)

	// Wiki write proxies (need auth + token forwarding)
	authed.Put("/galgame/:gid", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Post("/galgame/:gid/revert", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Post("/galgame/:gid/prs", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Put("/galgame/:gid/prs/:id/decline", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Post("/galgame/:gid/links", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/links", a.GalgameHandler.ProxyWriteWithToken("DELETE"))
	authed.Post("/galgame/:gid/aliases", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/aliases", a.GalgameHandler.ProxyWriteWithToken("DELETE"))
	authed.Delete("/galgame/:gid/contributors/:uid", a.GalgameHandler.ProxyWriteWithToken("DELETE"))
	authed.Put("/tag", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/official", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/engine", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Post("/series", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Post("/series/modal", a.GalgameHandler.ProxyWriteWithToken("POST"))
	authed.Put("/series/:id", a.GalgameHandler.ProxyWriteWithToken("PUT"))
	authed.Delete("/series/:id", a.GalgameHandler.ProxyWriteWithToken("DELETE"))

	// ── Toolset routes ──
	// Public (with optional auth for practicality "mine" field)
	optAuth.Get("/toolset", a.ToolsetHandler.GetList)
	optAuth.Get("/toolset/:id", a.ToolsetHandler.GetDetail)
	optAuth.Get("/toolset/:id/practicality", a.ToolsetHandler.GetPracticality)
	optAuth.Get("/toolset/:id/comment", a.ToolsetHandler.GetComments)
	api.Get("/toolset/:id/resource/detail", a.ToolsetHandler.GetResourceDetail)

	// Authenticated
	authed.Post("/toolset", a.ToolsetHandler.Create)
	authed.Put("/toolset/:id", a.ToolsetHandler.Update)
	authed.Delete("/toolset/:id", a.ToolsetHandler.Delete)
	authed.Put("/toolset/:id/practicality", a.ToolsetHandler.UpsertPracticality)
	authed.Post("/toolset/:id/comment", a.ToolsetHandler.CreateComment)
	authed.Put("/toolset/:id/comment", a.ToolsetHandler.UpdateComment)
	authed.Delete("/toolset/:id/comment", a.ToolsetHandler.DeleteComment)
	authed.Post("/toolset/:id/resource", a.ToolsetHandler.CreateResource)
	authed.Put("/toolset/:id/resource", a.ToolsetHandler.UpdateResource)
	authed.Delete("/toolset/:id/resource", a.ToolsetHandler.DeleteResource)
	authed.Post("/toolset/:id/upload/small", a.ToolsetHandler.UploadSmall)
	authed.Post("/toolset/:id/upload/large", a.ToolsetHandler.UploadLarge)
	authed.Post("/toolset/:id/upload/complete", a.ToolsetHandler.UploadComplete)
	authed.Post("/toolset/:id/upload/abort", a.ToolsetHandler.UploadAbort)

	// Local galgame interactions (no wiki service call)
	authed.Put("/galgame/:gid/like", a.GalgameHandler.ToggleLike)
	authed.Put("/galgame/:gid/favorite", a.GalgameHandler.ToggleFavorite)
	optAuth.Get("/galgame/:gid/comment", a.GalgameHandler.GetComments)
	authed.Post("/galgame/:gid/comment", a.GalgameHandler.CreateComment)
	authed.Delete("/galgame/:gid/comment", a.GalgameHandler.DeleteComment)
	authed.Put("/galgame/:gid/comment/like", a.GalgameHandler.ToggleCommentLike)
}
