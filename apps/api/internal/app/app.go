package app

import (
	"log/slog"

	adminHandler "kun-galgame-api/internal/admin/handler"
	"kun-galgame-api/internal/common"
	"kun-galgame-api/internal/infrastructure/cache"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/mail"
	"kun-galgame-api/internal/infrastructure/storage"
	msgHandler "kun-galgame-api/internal/message/handler"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	topicHandler "kun-galgame-api/internal/topic/handler"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/user/handler"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber  *fiber.App
	DB     *gorm.DB
	Redis  *redis.Client
	S3     *storage.S3Client
	Mailer *mail.Mailer
	Config *config.Config

	// Handlers
	OAuthHandler   *handler.OAuthHandler
	UserHandler    *handler.UserHandler
	HomeHandler    *common.HomeHandler
	TopicHandler   *topicHandler.TopicHandler
	ReplyHandler   *topicHandler.ReplyHandler
	PollHandler    *topicHandler.PollHandler
	MessageHandler *msgHandler.MessageHandler
	AdminHandler   *adminHandler.AdminHandler
	RankingHandler *common.RankingHandler
	SectionHandler *common.SectionHandler
}

func New(cfg *config.Config) *App {
	// Infrastructure
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	s3Client := storage.NewS3(cfg.S3)
	mailer := mail.NewMailer(cfg.Mail)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	messageRepository := msgRepo.NewMessageRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, rdb, cfg.OAuth)
	userService := service.NewUserService(userRepo, rdb)
	messageSvc := msgService.NewMessageService(messageRepository)

	// Topic
	topicRepository := topicRepo.NewTopicRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	pollRepository := topicRepo.NewPollRepository(db)
	topicSvc := topicService.NewTopicService(topicRepository, rdb)
	replySvc := topicService.NewReplyService(replyRepository, topicRepository, rdb)
	commentSvc := topicService.NewCommentService(replyRepository, rdb)
	pollSvc := topicService.NewPollService(pollRepository, topicRepository, rdb)
	topicHdl := topicHandler.NewTopicHandler(topicSvc)
	replyHdl := topicHandler.NewReplyHandler(replySvc, commentSvc)
	pollHdl := topicHandler.NewPollHandler(pollSvc)

	// Handlers
	oauthHandler := handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod")
	userHandler := handler.NewUserHandler(userService)
	homeHandler := common.NewHomeHandler(db)
	messageHdl := msgHandler.NewMessageHandler(messageSvc)
	adminHdl := adminHandler.NewAdminHandler(db, rdb)
	rankingHdl := common.NewRankingHandler(db)
	sectionHdl := common.NewSectionHandler(db)

	// Fiber
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: globalErrorHandler,
		BodyLimit:    10 * 1024 * 1024,
	})
	fiberApp.Use(recover.New())

	app := &App{
		Fiber:          fiberApp,
		DB:             db,
		Redis:          rdb,
		S3:             s3Client,
		Mailer:         mailer,
		Config:         cfg,
		OAuthHandler:   oauthHandler,
		UserHandler:    userHandler,
		HomeHandler:    homeHandler,
		TopicHandler:   topicHdl,
		ReplyHandler:   replyHdl,
		PollHandler:    pollHdl,
		MessageHandler: messageHdl,
		AdminHandler:   adminHdl,
		RankingHandler: rankingHdl,
		SectionHandler: sectionHdl,
	}

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
