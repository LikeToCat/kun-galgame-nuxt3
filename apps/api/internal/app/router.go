package app

import (
	"time"

	"kun-galgame-api/internal/middleware"

	fiberCors "github.com/gofiber/fiber/v2/middleware/cors"
)

func (a *App) setupRoutes() {
	a.Fiber.Use(fiberCors.New(middleware.CORS(a.Config.CORS.AllowOrigins)))

	api := a.Fiber.Group("/api")

	// ════════════════════════════════════════════
	// PUBLIC routes (no auth required)
	// Must be registered BEFORE authed group to avoid
	// Fiber Group("") middleware intercepting them.
	// ════════════════════════════════════════════

	api.Get("/home", a.HomeHandler.GetHome)

	// Auth (public)
	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)
	auth.Post("/email/code/reset", a.OAuthHandler.SendResetEmailCode)

	// User (authenticated, fixed paths — registered before :uid to avoid conflicts)
	userAuth := middleware.Auth(a.Redis, a.OAuthClient)
	checkInRL := middleware.RateLimit(a.Redis, "checkin", 1, 24*time.Hour)
	usernameRL := middleware.RateLimit(a.Redis, "username", 3, time.Hour)
	emailRL := middleware.RateLimit(a.Redis, "email", 3, time.Hour)
	avatarRL := middleware.RateLimit(a.Redis, "avatar", 5, time.Hour)
	api.Post("/user/check-in", userAuth, checkInRL, a.UserHandler.CheckIn)
	api.Put("/user/bio", userAuth, a.UserHandler.UpdateBio)
	api.Put("/user/username", userAuth, usernameRL, a.UserHandler.UpdateUsername)
	api.Put("/user/email", userAuth, emailRL, a.UserHandler.UpdateEmail)
	api.Get("/user/email", userAuth, a.UserHandler.GetEmail)
	api.Get("/user/status", userAuth, a.UserHandler.GetStatus)
	api.Post("/user/avatar", userAuth, avatarRL, a.UserHandler.UploadAvatar)

	// User (public, parameterized — AFTER fixed paths)
	api.Get("/user/:uid/floating", a.UserHandler.GetFloatingCard)
	api.Get("/user/:uid", a.UserHandler.GetProfile)
	api.Get("/user/:uid/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:uid/topics", a.UserHandler.GetUserTopics)
	api.Get("/user/:uid/replies", a.UserHandler.GetUserReplies)
	api.Get("/user/:uid/comments", a.UserHandler.GetUserComments)
	api.Get("/user/:uid/resources", a.UserHandler.GetUserResources)
	api.Get("/user/:uid/ratings", a.UserHandler.GetUserRatings)

	// Ranking (public)
	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	// Section & Category (public)
	api.Get("/section", a.SectionHandler.GetSectionTopics)
	api.Get("/category", a.SectionHandler.GetCategories)

	// Doc (public reads)
	api.Get("/doc/article", a.DocArticleHandler.GetArticles)
	api.Get("/doc/article/:slug", a.DocArticleHandler.GetArticleBySlug)
	api.Get("/doc/category", a.DocCategoryHandler.GetCategories)
	api.Get("/doc/tag", a.DocTagHandler.GetTags)

	// Website (public reads)
	api.Get("/website-category/:name", a.WebsiteCategoryHandler.GetWebsiteCategory)
	api.Get("/website-tag", a.WebsiteTagHandler.GetWebsiteTags)
	api.Get("/website-tag/:name", a.WebsiteTagHandler.GetWebsiteTagDetail)

	// Update (public reads)
	api.Get("/update/history", a.UpdateHandler.GetHistory)
	api.Get("/update/todo", a.UpdateHandler.GetTodos)

	// Admin setting (public read)
	api.Get("/admin/setting/register", a.AdminSettingHandler.GetRegisterSetting)

	// Message (public)
	api.Get("/message/admin", a.MessageHandler.GetSystemMessages)

	// Activity (public)
	api.Get("/activity", a.ActivityHandler.GetActivity)
	api.Get("/activity/timeline", a.ActivityHandler.GetTimeline)

	// Galgame rating (public)
	api.Get("/galgame-rating/all", a.GalgameRatingHandler.GetAllRatings)
	api.Get("/galgame-rating/:id", a.GalgameRatingHandler.GetRatingDetail)

	// Resource topics (public, same as topic but filtered to resource sections)
	api.Get("/resource", a.TopicHandler.GetResourceList)

	// Search (public)
	api.Get("/search", a.SearchHandler.Search)

	// RSS (public)
	api.Get("/rss/topic", a.RSSHandler.GetTopicRSS)
	api.Get("/rss/galgame", a.RSSHandler.GetGalgameRSS)

	// Unmoe (public)
	api.Get("/unmoe", a.UnmoeHandler.GetLogs)

	// Toolset resource detail (public)
	api.Get("/toolset/:id/resource/detail", a.ToolsetResourceHandler.GetResourceDetail)

	// Galgame wiki proxies (public reads)
	api.Get("/galgame", a.GalgameHandler.GetList)
	api.Get("/galgame/check", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev/diff", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/prs", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/prs/:id", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/links", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/contributors", a.GalgameWikiHandler.ProxyGet)
	// NOTE: galgame detail sub-routes (/pr/all, /link/all, etc.) are
	// registered in the optAuth group below to avoid Fiber route shadowing
	// by /galgame/:gid.
	api.Get("/galgame-tag", a.GalgameEntityHandler.GetTagList)
	api.Get("/galgame-tag/search", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame-tag/multi", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame-tag/:name", a.GalgameEntityHandler.GetTagDetail)
	api.Get("/galgame-official", a.GalgameEntityHandler.GetOfficialList)
	api.Get("/galgame-official/search", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame-official/:name", a.GalgameEntityHandler.GetOfficialDetail)
	api.Get("/galgame-engine", a.GalgameEntityHandler.GetEngineList)
	api.Get("/galgame-engine/:name", a.GalgameEntityHandler.GetEngineDetail)
	api.Get("/galgame-series", a.GalgameEntityHandler.GetSeriesList)
	api.Get("/galgame-series/search", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame-series/:id", a.GalgameEntityHandler.GetSeriesDetail)
	api.Get("/galgame-resource", a.GalgameResourceHandler.GetResourceList)

	// ════════════════════════════════════════════
	// OPTIONAL AUTH routes (public but attach user if logged in)
	// ════════════════════════════════════════════

	optAuth := api.Group("", middleware.OptionalAuth(a.Redis, a.OAuthClient))
	optAuth.Get("/galgame-resource/:id/detail", a.GalgameResourceHandler.GetResourceDownloadDetail)
	optAuth.Get("/galgame-resource/:id/recommend", a.GalgameResourceHandler.GetRecommend)
	optAuth.Get("/galgame-resource/:id", a.GalgameResourceHandler.GetResourceDetail)

	// Topic (optional auth for interaction status)
	optAuth.Get("/topic", a.TopicHandler.GetList)
	optAuth.Get("/topic/:tid", a.TopicHandler.GetDetail)
	optAuth.Get("/topic/:tid/reply", a.ReplyHandler.GetReplies)
	optAuth.Get("/topic/:tid/reply/detail", a.ReplyHandler.GetReplyDetail)
	optAuth.Get("/topic/:tid/poll/topic", a.PollHandler.GetPollsByTopic)
	optAuth.Get("/topic/:tid/poll/log", a.PollHandler.GetVoteLog)

	// Galgame detail sub-routes (MUST be before /:gid to avoid shadowing)
	optAuth.Get("/galgame/:gid/resource/all", a.GalgameResourceHandler.GetGalgameResources)
	optAuth.Get("/galgame/:gid/comment/all", a.GalgameCommentHandler.GetComments)
	optAuth.Get("/galgame/:gid/pr/all", a.GalgameWikiHandler.GetGalgamePRs)
	optAuth.Get("/galgame/:gid/link/all", a.GalgameWikiHandler.GetGalgameLinks)
	optAuth.Get("/galgame/:gid/history/all", a.GalgameWikiHandler.GetGalgameHistory)
	optAuth.Get("/galgame/:gid", a.GalgameHandler.GetDetail)

	// Website (optional auth for like/favorite status)
	optAuth.Get("/website", a.WebsiteHandler.GetWebsites)
	optAuth.Get("/website/:domain/comment", a.WebsiteCommentHandler.GetComments)
	optAuth.Get("/website/:domain", a.WebsiteHandler.GetWebsiteDetail)

	// Toolset (optional auth for practicality "mine" field)
	optAuth.Get("/toolset", a.ToolsetHandler.GetList)
	optAuth.Get("/toolset/:id", a.ToolsetHandler.GetDetail)
	optAuth.Get("/toolset/:id/practicality", a.ToolsetPracticalityHandler.GetPracticality)
	optAuth.Get("/toolset/:id/comment/all", a.ToolsetCommentHandler.GetComments)

	// ════════════════════════════════════════════
	// AUTHENTICATED routes (require valid session)
	// ════════════════════════════════════════════

	authed := api.Group("", middleware.Auth(a.Redis, a.OAuthClient))
	authed.Get("/auth/me", a.OAuthHandler.Me)

	// Topic (authenticated)
	authed.Post("/topic", a.TopicHandler.Create)
	authed.Put("/topic/:tid", a.TopicHandler.Update)
	authed.Put("/topic/:tid/like", a.TopicHandler.ToggleLike)
	authed.Put("/topic/:tid/dislike", a.TopicHandler.ToggleDislike)
	authed.Put("/topic/:tid/upvote", a.TopicHandler.Upvote)
	authed.Put("/topic/:tid/favorite", a.TopicHandler.ToggleFavorite)
	authed.Put("/topic/:tid/hide", a.TopicHandler.ToggleHide)
	authed.Put("/topic/:tid/best-answer", a.TopicHandler.SetBestAnswer)

	// Reply (authenticated)
	authed.Post("/topic/:tid/reply", a.ReplyHandler.CreateReply)
	authed.Put("/topic/:tid/reply", a.ReplyHandler.UpdateReply)
	authed.Delete("/topic/:tid/reply", a.ReplyHandler.DeleteReply)
	authed.Put("/topic/:tid/reply/like", a.ReplyHandler.ToggleReplyLike)
	authed.Put("/topic/:tid/reply/dislike", a.ReplyHandler.ToggleReplyDislike)
	authed.Put("/topic/:tid/reply/pin", a.ReplyHandler.PinReply)

	// Comment (authenticated)
	authed.Post("/topic/:tid/comment", a.TopicCommentHandler.CreateComment)
	authed.Put("/topic/:tid/comment/like", a.TopicCommentHandler.ToggleCommentLike)
	authed.Delete("/topic/:tid/comment", a.TopicCommentHandler.DeleteComment)

	// Poll (authenticated)
	authed.Post("/topic/:tid/poll", a.PollHandler.CreatePoll)
	authed.Put("/topic/:tid/poll", a.PollHandler.UpdatePoll)
	authed.Delete("/topic/:tid/poll", a.PollHandler.DeletePoll)
	authed.Post("/topic/:tid/poll/vote", a.PollHandler.Vote)

	// Message (authenticated)
	authed.Get("/message", a.MessageHandler.GetMessages)
	authed.Delete("/message/:id", a.MessageHandler.DeleteMessage)
	authed.Put("/message/system/read", a.MessageHandler.MarkAllRead)
	authed.Put("/message/admin/read", a.MessageHandler.MarkAdminRead)
	authed.Get("/message/nav/system", a.MessageHandler.GetNavSummary)
	authed.Get("/message/nav/contact", a.MessageChatHandler.GetNavContact)
	authed.Get("/message/chat/history", a.MessageChatHandler.GetChatHistory)
	authed.Post("/message/chat/send", a.MessageChatHandler.SendChatMessage)

	// Image upload (authenticated)
	authed.Post("/image/topic", a.ImageHandler.UploadTopicImage)

	// Report (authenticated)
	authed.Post("/report/submit", a.ReportHandler.SubmitReport)

	// Website interactions (authenticated)
	authed.Put("/website/:domain/like", a.WebsiteHandler.ToggleLike)
	authed.Put("/website/:domain/favorite", a.WebsiteHandler.ToggleFavorite)
	authed.Post("/website/:domain/comment", a.WebsiteCommentHandler.CreateComment)
	authed.Delete("/website/:domain/comment", a.WebsiteCommentHandler.DeleteComment)

	// Galgame interactions (authenticated, local)
	authed.Put("/galgame/:gid/like", a.GalgameHandler.ToggleLike)
	authed.Put("/galgame/:gid/favorite", a.GalgameHandler.ToggleFavorite)
	authed.Post("/galgame/:gid/comment", a.GalgameCommentHandler.CreateComment)
	authed.Delete("/galgame/:gid/comment", a.GalgameCommentHandler.DeleteComment)
	authed.Put("/galgame/:gid/comment/like", a.GalgameCommentHandler.ToggleCommentLike)

	// Galgame resource (authenticated, local)
	authed.Post("/galgame/:gid/resource", a.GalgameResourceHandler.CreateResource)
	authed.Put("/galgame/:gid/resource", a.GalgameResourceHandler.UpdateResource)
	authed.Delete("/galgame/:gid/resource", a.GalgameResourceHandler.DeleteResource)
	authed.Put("/galgame/:gid/resource/like", a.GalgameResourceHandler.ToggleLike)
	authed.Put("/galgame/:gid/resource/valid", a.GalgameResourceHandler.MarkValid)
	authed.Put("/galgame/:gid/resource/expired", a.GalgameResourceHandler.MarkExpired)

	// Galgame rating (authenticated, local)
	authed.Post("/galgame-rating", a.GalgameRatingHandler.CreateRating)
	authed.Put("/galgame-rating/:id", a.GalgameRatingHandler.UpdateRating)
	authed.Delete("/galgame-rating/:id", a.GalgameRatingHandler.DeleteRating)
	authed.Put("/galgame-rating/:id/like", a.GalgameRatingHandler.ToggleLike)
	authed.Post("/galgame-rating/:id/comment", a.GalgameRatingHandler.CreateComment)
	authed.Put("/galgame-rating/:id/comment", a.GalgameRatingHandler.UpdateComment)
	authed.Delete("/galgame-rating/:id/comment", a.GalgameRatingHandler.DeleteComment)

	// Galgame wiki writes (authenticated + token forwarding)
	authed.Post("/galgame", a.GalgameHandler.Create)
	authed.Put("/galgame/:gid", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame/:gid/prs/:id/merge", a.GalgameHandler.MergePR)
	authed.Post("/galgame/:gid/revert", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Post("/galgame/:gid/prs", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Put("/galgame/:gid/prs/:id/decline", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Post("/galgame/:gid/links", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/links", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Post("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Delete("/galgame/:gid/contributors/:uid", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Put("/galgame-tag", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame-official", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame-engine", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Post("/galgame-series", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Post("/galgame-series/modal", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Put("/galgame-series/:id", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Delete("/galgame-series/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))

	// Toolset (authenticated)
	authed.Post("/toolset", a.ToolsetHandler.Create)
	authed.Put("/toolset/:id", a.ToolsetHandler.Update)
	authed.Delete("/toolset/:id", a.ToolsetHandler.Delete)
	authed.Put("/toolset/:id/practicality", a.ToolsetPracticalityHandler.UpsertPracticality)
	authed.Post("/toolset/:id/comment", a.ToolsetCommentHandler.CreateComment)
	authed.Put("/toolset/:id/comment", a.ToolsetCommentHandler.UpdateComment)
	authed.Delete("/toolset/:id/comment", a.ToolsetCommentHandler.DeleteComment)
	authed.Post("/toolset/:id/resource", a.ToolsetResourceHandler.CreateResource)
	authed.Put("/toolset/:id/resource", a.ToolsetResourceHandler.UpdateResource)
	authed.Delete("/toolset/:id/resource", a.ToolsetResourceHandler.DeleteResource)
	authed.Post("/toolset/:id/upload/small", a.ToolsetUploadHandler.UploadSmall)
	authed.Post("/toolset/:id/upload/large", a.ToolsetUploadHandler.UploadLarge)
	authed.Post("/toolset/:id/upload/complete", a.ToolsetUploadHandler.UploadComplete)
	authed.Post("/toolset/:id/upload/abort", a.ToolsetUploadHandler.UploadAbort)

	// ════════════════════════════════════════════
	// ADMIN routes (require role >= 2 or 3)
	// ════════════════════════════════════════════

	admin := authed.Group("", middleware.RequireRole(3))
	admin.Put("/user/:uid/ban", a.UserHandler.BanUser)
	admin.Delete("/user/:uid", a.UserHandler.DeleteUser)
	admin.Get("/admin/overview/all", a.AdminOverviewHandler.GetOverview)
	admin.Get("/admin/overview/stats", a.AdminOverviewHandler.GetStats)
	admin.Put("/admin/setting/register", a.AdminSettingHandler.ToggleRegisterSetting)

	adminRead := authed.Group("", middleware.RequireRole(2))
	adminRead.Get("/admin/user", a.AdminUserHandler.GetUserList)
	adminRead.Get("/admin/user/search", a.AdminUserHandler.SearchUsers)

	// Doc admin (role >= 2)
	docAdmin := authed.Group("", middleware.RequireRole(2))
	docAdmin.Post("/doc/article", a.DocArticleHandler.CreateArticle)
	docAdmin.Put("/doc/article", a.DocArticleHandler.UpdateArticle)
	docAdmin.Delete("/doc/article", a.DocArticleHandler.DeleteArticle)
	docAdmin.Post("/doc/category", a.DocCategoryHandler.CreateCategory)
	docAdmin.Put("/doc/category", a.DocCategoryHandler.UpdateCategory)
	docAdmin.Delete("/doc/category", a.DocCategoryHandler.DeleteCategory)
	docAdmin.Post("/doc/tag", a.DocTagHandler.CreateTag)
	docAdmin.Put("/doc/tag", a.DocTagHandler.UpdateTag)
	docAdmin.Delete("/doc/tag", a.DocTagHandler.DeleteTag)

	// Website admin (role >= 2)
	wsAdmin := authed.Group("", middleware.RequireRole(2))
	wsAdmin.Post("/website", a.WebsiteHandler.CreateWebsite)
	wsAdmin.Put("/website/:domain", a.WebsiteHandler.UpdateWebsite)
	wsAdmin.Delete("/website/:domain", a.WebsiteHandler.DeleteWebsite)
	wsAdmin.Put("/website-category", a.WebsiteCategoryHandler.UpdateWebsiteCategory)
	wsAdmin.Post("/website-tag", a.WebsiteTagHandler.CreateWebsiteTag)
	wsAdmin.Put("/website-tag", a.WebsiteTagHandler.UpdateWebsiteTag)
	wsAdmin.Delete("/website-tag", a.WebsiteTagHandler.DeleteWebsiteTag)

	// Update admin (role >= 2)
	updateAdmin := authed.Group("", middleware.RequireRole(2))
	updateAdmin.Post("/update/history", a.UpdateHandler.CreateHistory)
	updateAdmin.Put("/update/history", a.UpdateHandler.UpdateHistory)
	updateAdmin.Delete("/update/history", a.UpdateHandler.DeleteHistory)
	updateAdmin.Post("/update/todo", a.UpdateHandler.CreateTodo)
	updateAdmin.Put("/update/todo", a.UpdateHandler.UpdateTodo)
	updateAdmin.Delete("/update/todo", a.UpdateHandler.DeleteTodo)
}
