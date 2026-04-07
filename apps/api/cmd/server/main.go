package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kun-galgame-api/internal/app"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env (ignore error in production where env vars are set externally)
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.Server.Mode)

	application := app.New(cfg)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("正在关闭服务器...")
		_ = application.Fiber.Shutdown()
	}()

	addr := ":" + cfg.Server.Port
	slog.Info("服务器启动", "addr", addr)
	if err := application.Fiber.Listen(addr); err != nil {
		slog.Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}
