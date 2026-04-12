package middleware

import (
	"fmt"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimit creates a per-user rate limiting middleware using Redis.
// maxRequests is the maximum number of requests allowed within the window.
func RateLimit(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}

		key := fmt.Sprintf("ratelimit:%s:%d", prefix, user.UID)
		ctx := c.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(maxRequests) {
			return response.Error(c, errors.ErrBadRequest("操作过于频繁，请稍后再试"))
		}

		return c.Next()
	}
}
