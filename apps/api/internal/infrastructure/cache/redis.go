package cache

import (
	"context"
	"fmt"
	"log/slog"

	"kun-galgame-api/pkg/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("连接 Redis 失败: %v", err))
	}

	slog.Info("Redis 连接成功")
	return client
}
