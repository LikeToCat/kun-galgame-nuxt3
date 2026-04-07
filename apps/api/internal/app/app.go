package app

import (
	"kun-galgame-api/internal/common"
	"kun-galgame-api/internal/infrastructure/cache"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/mail"
	"kun-galgame-api/internal/infrastructure/storage"
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
	OAuthHandler *handler.OAuthHandler
	UserHandler  *handler.UserHandler
	HomeHandler  *common.HomeHandler
}

func New(cfg *config.Config) *App {
	// Infrastructure
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	s3Client := storage.NewS3(cfg.S3)
	mailer := mail.NewMailer(cfg.Mail)

	// Repositories
	userRepo := repository.NewUserRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, rdb, cfg.OAuth)
	userService := service.NewUserService(userRepo, rdb)

	// Handlers
	oauthHandler := handler.NewOAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	homeHandler := common.NewHomeHandler(db)

	// Fiber
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: globalErrorHandler,
		BodyLimit:    10 * 1024 * 1024,
	})
	fiberApp.Use(recover.New())

	app := &App{
		Fiber:        fiberApp,
		DB:           db,
		Redis:        rdb,
		S3:           s3Client,
		Mailer:       mailer,
		Config:       cfg,
		OAuthHandler: oauthHandler,
		UserHandler:  userHandler,
		HomeHandler:  homeHandler,
	}

	app.setupRoutes()
	return app
}

func globalErrorHandler(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}
