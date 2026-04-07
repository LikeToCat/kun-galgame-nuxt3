package main

import (
	"log/slog"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	slog.Info("开始数据库迁移...")

	// Only migrate new tables that don't exist in Prisma schema.
	// Existing tables are managed by the Prisma schema and should NOT
	// be auto-migrated here to avoid conflicts.
	err := db.AutoMigrate(
		&model.OAuthAccount{},
	)
	if err != nil {
		slog.Error("数据库迁移失败", "error", err)
		return
	}

	slog.Info("数据库迁移完成")
}
