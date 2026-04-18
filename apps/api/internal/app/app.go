package app

import (
	"log/slog"

	activityHandler "kun-galgame-api/internal/activity/handler"
	activityRepo "kun-galgame-api/internal/activity/repository"
	activityService "kun-galgame-api/internal/activity/service"
	adminHandler "kun-galgame-api/internal/admin/handler"
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	docHandler "kun-galgame-api/internal/doc/handler"
	docRepo "kun-galgame-api/internal/doc/repository"
	docService "kun-galgame-api/internal/doc/service"
	galgameClient "kun-galgame-api/internal/galgame/client"
	galgameHandler "kun-galgame-api/internal/galgame/handler"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	homeHandler "kun-galgame-api/internal/home/handler"
	homeRepo "kun-galgame-api/internal/home/repository"
	homeService "kun-galgame-api/internal/home/service"
	imageHandler "kun-galgame-api/internal/image/handler"
	imageRepo "kun-galgame-api/internal/image/repository"
	imageService "kun-galgame-api/internal/image/service"
	toolsetHandler "kun-galgame-api/internal/toolset/handler"
	toolsetRepo "kun-galgame-api/internal/toolset/repository"
	toolsetService "kun-galgame-api/internal/toolset/service"
	cronPkg "kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/infrastructure/cache"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/mail"
	"kun-galgame-api/internal/infrastructure/storage"
	msgHandler "kun-galgame-api/internal/message/handler"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	rankingHandler "kun-galgame-api/internal/ranking/handler"
	rankingRepo "kun-galgame-api/internal/ranking/repository"
	rankingService "kun-galgame-api/internal/ranking/service"
	reportHandler "kun-galgame-api/internal/report/handler"
	reportRepo "kun-galgame-api/internal/report/repository"
	rssHandler "kun-galgame-api/internal/rss/handler"
	rssRepo "kun-galgame-api/internal/rss/repository"
	searchHandler "kun-galgame-api/internal/search/handler"
	searchRepo "kun-galgame-api/internal/search/repository"
	searchService "kun-galgame-api/internal/search/service"
	sectionHandler "kun-galgame-api/internal/section/handler"
	sectionRepo "kun-galgame-api/internal/section/repository"
	sectionService "kun-galgame-api/internal/section/service"
	topicHandler "kun-galgame-api/internal/topic/handler"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	unmoeHandler "kun-galgame-api/internal/unmoe/handler"
	unmoeRepo "kun-galgame-api/internal/unmoe/repository"
	updateHandler "kun-galgame-api/internal/update/handler"
	updateRepo "kun-galgame-api/internal/update/repository"
	"kun-galgame-api/internal/user/handler"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	websiteHandler "kun-galgame-api/internal/website/handler"
	websiteRepo "kun-galgame-api/internal/website/repository"
	websiteService "kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber       *fiber.App
	DB          *gorm.DB
	Redis       *redis.Client
	S3          *storage.S3Client
	Mailer      *mail.Mailer
	Config      *config.Config
	OAuthClient *oauth.Client

	// Handlers
	OAuthHandler   *handler.OAuthHandler
	UserHandler    *handler.UserHandler
	HomeHandler    *homeHandler.HomeHandler
	TopicHandler        *topicHandler.TopicHandler
	ReplyHandler        *topicHandler.ReplyHandler
	TopicCommentHandler *topicHandler.CommentHandler
	PollHandler         *topicHandler.PollHandler
	MessageHandler     *msgHandler.MessageHandler
	MessageChatHandler *msgHandler.ChatHandler
	AdminOverviewHandler *adminHandler.OverviewHandler
	AdminSettingHandler  *adminHandler.SettingHandler
	AdminUserHandler     *adminHandler.UserHandler
	RankingHandler *rankingHandler.RankingHandler
	SectionHandler *sectionHandler.SectionHandler
	DocArticleHandler  *docHandler.ArticleHandler
	DocCategoryHandler *docHandler.CategoryHandler
	DocTagHandler      *docHandler.TagHandler
	WebsiteHandler         *websiteHandler.WebsiteHandler
	WebsiteCommentHandler  *websiteHandler.CommentHandler
	WebsiteCategoryHandler *websiteHandler.CategoryHandler
	WebsiteTagHandler      *websiteHandler.TagHandler
	UpdateHandler    *updateHandler.UpdateHandler
	UnmoeHandler     *unmoeHandler.UnmoeHandler
	ReportHandler    *reportHandler.ReportHandler
	RSSHandler       *rssHandler.RSSHandler
	GalgameHandler         *galgameHandler.GalgameHandler
	GalgameCommentHandler  *galgameHandler.CommentHandler
	GalgameResourceHandler *galgameHandler.ResourceHandler
	GalgameRatingHandler   *galgameHandler.RatingHandler
	GalgameEntityHandler   *galgameHandler.EntityHandler
	GalgameWikiHandler     *galgameHandler.WikiHandler
	ActivityHandler  *activityHandler.ActivityHandler
	ImageHandler     *imageHandler.ImageHandler
	SearchHandler    *searchHandler.SearchHandler
	ToolsetHandler             *toolsetHandler.ToolsetHandler
	ToolsetPracticalityHandler *toolsetHandler.PracticalityHandler
	ToolsetCommentHandler      *toolsetHandler.CommentHandler
	ToolsetResourceHandler     *toolsetHandler.ResourceHandler
	ToolsetUploadHandler       *toolsetHandler.UploadHandler
	CronStop         func()
}

func New(cfg *config.Config) *App {
	// Infrastructure
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	s3Client := storage.NewS3(cfg.S3)
	mailer := mail.NewMailer(cfg.Mail)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	userStatsRepo := repository.NewUserStatsRepository(db)
	userContentRepo := repository.NewUserContentRepository(db)
	userBriefRepo := repository.NewUserBriefRepository(db)
	messageRepository := msgRepo.NewMessageRepository(db)
	chatRepository := msgRepo.NewChatRepository(db)

	// Galgame wiki client (shared — user service needs it too).
	gc := galgameClient.NewGalgameClient(cfg.GalgameWiki.BaseURL)

	// OAuth client (used by auth service).
	oauthClient := oauth.NewClient(cfg.OAuth)

	// Services
	authService := service.NewAuthService(userRepo, rdb, oauthClient, mailer)
	userService := service.NewUserService(userRepo, userStatsRepo, userBriefRepo, rdb, gc)
	userContentService := service.NewUserContentService(userContentRepo, userBriefRepo, gc)
	messageSvc := msgService.NewMessageService(messageRepository)
	chatSvc := msgService.NewChatService(chatRepository)

	// Topic
	topicRepository := topicRepo.NewTopicRepository(db)
	topicListRepo := topicRepo.NewTopicListRepository(db)
	topicTaxonomyRepo := topicRepo.NewTopicTaxonomyRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	topicCommentRepo := topicRepo.NewCommentRepository(db)
	pollRepository := topicRepo.NewPollRepository(db)
	topicSvc := topicService.NewTopicService(topicRepository, topicListRepo, topicTaxonomyRepo, rdb)
	topicWriteSvc := topicService.NewTopicWriteService(topicRepository, topicTaxonomyRepo, rdb)
	replySvc := topicService.NewReplyService(replyRepository, topicCommentRepo, topicRepository, rdb)
	commentSvc := topicService.NewCommentService(replyRepository, topicCommentRepo, rdb)
	pollSvc := topicService.NewPollService(pollRepository, topicRepository, rdb)

	// Galgame
	galgameCommentRepo := galgameRepo.NewCommentRepository(db)
	galgameCommentSvc := galgameService.NewCommentService(galgameCommentRepo)
	galgameResourceRepo := galgameRepo.NewResourceRepository(db)
	galgameResourceSvc := galgameService.NewResourceService(galgameResourceRepo, gc)
	galgameRatingRepo := galgameRepo.NewRatingRepository(db)
	galgameRatingSvc := galgameService.NewRatingService(galgameRatingRepo, gc)
	galgameLocalRepo := galgameRepo.NewGalgameRepository(db)
	galgameInteractionRepo := galgameRepo.NewGalgameInteractionRepository(db)
	galgameListRepo := galgameRepo.NewGalgameListRepository(db)
	galgameResourceMetaRepo := galgameRepo.NewGalgameResourceMetaRepository(db)
	galgameDetailRatingRepo := galgameRepo.NewGalgameDetailRatingRepository(db)
	galgameEnricher := galgameService.NewGalgameEnricher(galgameLocalRepo)
	galgameSeriesSvc := galgameService.NewSeriesService(gc, galgameEnricher)
	galgameOfficialSvc := galgameService.NewOfficialService(gc, galgameEnricher)
	galgameEngineSvc := galgameService.NewEngineService(gc, galgameEnricher)
	galgameTagSvc := galgameService.NewTagService(gc, galgameEnricher)
	galgameWikiSvc := galgameService.NewWikiService(gc, galgameLocalRepo)
	galgameCoreSvc := galgameService.NewGalgameService(
		galgameLocalRepo, galgameInteractionRepo, galgameListRepo,
		galgameResourceMetaRepo, galgameDetailRatingRepo, gc,
	)

	// Website
	websiteRepository := websiteRepo.NewWebsiteRepository(db)
	websiteCategoryRepo := websiteRepo.NewCategoryRepository(db)
	websiteTagRepo := websiteRepo.NewTagRepository(db)
	websiteCommentRepo := websiteRepo.NewCommentRepository(db)
	websiteCoreSvc := websiteService.NewWebsiteService(
		websiteRepository, websiteCategoryRepo, websiteTagRepo, websiteCommentRepo,
	)
	websiteCommentSvc := websiteService.NewCommentService(websiteCommentRepo, websiteRepository)
	websiteCategorySvc := websiteService.NewCategoryService(websiteCategoryRepo, websiteRepository, websiteTagRepo)
	websiteTagSvc := websiteService.NewTagService(websiteTagRepo, websiteRepository, websiteCategoryRepo)

	// Admin
	adminOverviewRepo := adminRepo.NewOverviewRepository(db)
	adminSettingRepo := adminRepo.NewSettingRepository(rdb)
	adminUserRepo := adminRepo.NewUserRepository(db)
	adminOverviewSvc := adminService.NewOverviewService(adminOverviewRepo, gc)
	adminSettingSvc := adminService.NewSettingService(adminSettingRepo)
	adminUserSvc := adminService.NewUserService(adminUserRepo)

	// Doc
	docArticleRepo := docRepo.NewArticleRepository(db)
	docCategoryRepo := docRepo.NewCategoryRepository(db)
	docTagRepo := docRepo.NewTagRepository(db)
	docArticleSvc := docService.NewArticleService(docArticleRepo)
	docCategorySvc := docService.NewCategoryService(docCategoryRepo)
	docTagSvc := docService.NewTagService(docTagRepo)

	// Toolset
	toolsetRepository := toolsetRepo.NewToolsetRepository(db)
	toolsetResourceRepo := toolsetRepo.NewResourceRepository(db)
	toolsetCommentRepo := toolsetRepo.NewCommentRepository(db)
	toolsetPracticalityRepo := toolsetRepo.NewPracticalityRepository(db)
	toolsetPracticalitySvc := toolsetService.NewPracticalityService(toolsetPracticalityRepo)
	toolsetCommentSvc := toolsetService.NewCommentService(toolsetCommentRepo, toolsetRepository)
	toolsetResourceSvc := toolsetService.NewResourceService(toolsetResourceRepo, toolsetRepository, s3Client)
	toolsetUploadSvc := toolsetService.NewUploadService(s3Client, rdb, db)
	toolsetCoreSvc := toolsetService.NewToolsetService(
		toolsetRepository, toolsetResourceRepo, toolsetCommentRepo, toolsetPracticalityRepo,
		s3Client, toolsetPracticalitySvc, toolsetCommentSvc,
	)

	// Handlers
	app := &App{
		DB: db, Redis: rdb, S3: s3Client, Mailer: mailer, Config: cfg, OAuthClient: oauthClient,
		OAuthHandler:   handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod"),
		UserHandler:    handler.NewUserHandler(userService, userContentService),
		HomeHandler:    homeHandler.NewHomeHandler(homeService.NewHomeService(homeRepo.NewHomeRepository(db), gc)),
		TopicHandler:        topicHandler.NewTopicHandler(topicSvc, topicWriteSvc),
		ReplyHandler:        topicHandler.NewReplyHandler(replySvc),
		TopicCommentHandler: topicHandler.NewCommentHandler(commentSvc),
		PollHandler:         topicHandler.NewPollHandler(pollSvc),
		MessageHandler:     msgHandler.NewMessageHandler(messageSvc),
		MessageChatHandler: msgHandler.NewChatHandler(chatSvc),
		AdminOverviewHandler: adminHandler.NewOverviewHandler(adminOverviewSvc),
		AdminSettingHandler:  adminHandler.NewSettingHandler(adminSettingSvc),
		AdminUserHandler:     adminHandler.NewUserHandler(adminUserSvc),
		RankingHandler: rankingHandler.NewRankingHandler(rankingService.NewRankingService(rankingRepo.NewRankingRepository(db), gc)),
		SectionHandler: sectionHandler.NewSectionHandler(sectionService.NewSectionService(sectionRepo.NewSectionRepository(db))),
		DocArticleHandler:  docHandler.NewArticleHandler(docArticleSvc),
		DocCategoryHandler: docHandler.NewCategoryHandler(docCategorySvc),
		DocTagHandler:      docHandler.NewTagHandler(docTagSvc),
		WebsiteHandler:         websiteHandler.NewWebsiteHandler(websiteCoreSvc),
		WebsiteCommentHandler:  websiteHandler.NewCommentHandler(websiteCommentSvc),
		WebsiteCategoryHandler: websiteHandler.NewCategoryHandler(websiteCategorySvc),
		WebsiteTagHandler:      websiteHandler.NewTagHandler(websiteTagSvc),
		UpdateHandler:    updateHandler.NewUpdateHandler(updateRepo.NewUpdateRepository(db)),
		UnmoeHandler:     unmoeHandler.NewUnmoeHandler(unmoeRepo.NewUnmoeRepository(db)),
		ReportHandler:    reportHandler.NewReportHandler(reportRepo.NewReportRepository(db)),
		RSSHandler:       rssHandler.NewRSSHandler(rssRepo.NewRSSRepository(db), gc, userBriefRepo),
		GalgameHandler:         galgameHandler.NewGalgameHandler(galgameCoreSvc),
		GalgameCommentHandler:  galgameHandler.NewCommentHandler(galgameCommentSvc),
		GalgameResourceHandler: galgameHandler.NewResourceHandler(galgameResourceSvc),
		GalgameRatingHandler:   galgameHandler.NewRatingHandler(galgameRatingSvc),
		GalgameEntityHandler: galgameHandler.NewEntityHandler(
			galgameSeriesSvc, galgameOfficialSvc, galgameEngineSvc, galgameTagSvc,
		),
		GalgameWikiHandler: galgameHandler.NewWikiHandler(galgameWikiSvc),
		ActivityHandler:  activityHandler.NewActivityHandler(activityService.NewActivityService(activityRepo.NewActivityRepository(db), gc)),
		ImageHandler:     imageHandler.NewImageHandler(imageService.NewImageService(imageRepo.NewImageRepository(db), s3Client)),
		SearchHandler:    searchHandler.NewSearchHandler(searchService.NewSearchService(searchRepo.NewSearchRepository(db))),
		ToolsetHandler:             toolsetHandler.NewToolsetHandler(toolsetCoreSvc),
		ToolsetPracticalityHandler: toolsetHandler.NewPracticalityHandler(toolsetPracticalitySvc),
		ToolsetCommentHandler:      toolsetHandler.NewCommentHandler(toolsetCommentSvc),
		ToolsetResourceHandler:     toolsetHandler.NewResourceHandler(toolsetResourceSvc),
		ToolsetUploadHandler:       toolsetHandler.NewUploadHandler(toolsetUploadSvc),
		CronStop:         cronPkg.Start(db, rdb),
	}

	// Fiber
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: globalErrorHandler,
		BodyLimit:    10 * 1024 * 1024,
	})
	fiberApp.Use(recover.New())
	app.Fiber = fiberApp

	app.setupRoutes()
	return app
}

func globalErrorHandler(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	slog.Error("未处理的错误", "error", err.Error(), "path", c.Path(), "method", c.Method())
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}
