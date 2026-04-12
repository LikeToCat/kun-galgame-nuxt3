package cron

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Start creates and starts all scheduled tasks. Returns a stop function.
func Start(db *gorm.DB, rdb *redis.Client) func() {
	c := cron.New()

	// Daily reset at midnight: clear daily check-in, image count, toolset upload count
	c.AddFunc("0 0 * * *", func() {
		resetDaily(db)
	})

	// Hourly: clean up abandoned toolset upload caches
	c.AddFunc("0 * * * *", func() {
		cleanupUploadCache(rdb)
	})

	c.Start()
	slog.Info("定时任务已启动")

	return func() {
		ctx := c.Stop()
		<-ctx.Done()
		slog.Info("定时任务已停止")
	}
}

// resetDaily resets all users' daily counters to 0.
func resetDaily(db *gorm.DB) {
	result := db.Exec(`
		UPDATE "user" SET
			daily_check_in = 0,
			daily_image_count = 0,
			daily_toolset_upload_count = 0
		WHERE daily_check_in != 0
		   OR daily_image_count != 0
		   OR daily_toolset_upload_count != 0
	`)
	if result.Error != nil {
		slog.Error("每日重置失败", "error", result.Error)
		return
	}
	slog.Info("每日重置完成", "affected", result.RowsAffected)
}

// cleanupUploadCache removes abandoned toolset upload artifacts from Redis.
// S3 cleanup is skipped here since S3 lifecycle rules handle orphaned objects.
func cleanupUploadCache(rdb *redis.Client) {
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "toolset:upload:*").Result()
	if err != nil {
		slog.Error("扫描上传缓存失败", "error", err)
		return
	}

	if len(keys) == 0 {
		return
	}

	deleted := 0
	for _, key := range keys {
		ttl, _ := rdb.TTL(ctx, key).Result()
		// Only delete keys with no TTL (stuck) or already expired
		if ttl <= 0 {
			rdb.Del(ctx, key)
			deleted++
		}
	}

	if deleted > 0 {
		slog.Info("清理上传缓存完成", "deleted", deleted)
	}
}
