package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// SettingRepository owns the Redis-backed admin flags.
type SettingRepository struct {
	rdb *redis.Client
}

func NewSettingRepository(rdb *redis.Client) *SettingRepository {
	return &SettingRepository{rdb: rdb}
}

const redisDisableRegisterKey = "kun:disable_register"

// GetRegisterDisabled returns true when the "disable register" flag is set.
func (r *SettingRepository) GetRegisterDisabled(ctx context.Context) bool {
	val, err := r.rdb.Get(ctx, redisDisableRegisterKey).Result()
	return err == nil && val == "1"
}

// ToggleRegisterDisabled flips the "disable register" flag and returns the
// new disabled state.
func (r *SettingRepository) ToggleRegisterDisabled(ctx context.Context) bool {
	val, _ := r.rdb.Get(ctx, redisDisableRegisterKey).Result()
	if val == "1" {
		r.rdb.Del(ctx, redisDisableRegisterKey)
		return false
	}
	r.rdb.Set(ctx, redisDisableRegisterKey, "1", 0)
	return true
}
