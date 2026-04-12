package handler

import (
	"context"
	"fmt"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const redisDisableRegisterKey = "kun:disable_register"

type AdminHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewAdminHandler(db *gorm.DB, rdb *redis.Client) *AdminHandler {
	return &AdminHandler{db: db, rdb: rdb}
}

// ──────────────────────────────────────────
// Overview
// ──────────────────────────────────────────

type overviewItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// GetOverview returns counts for all major models.
// GET /api/admin/overview/all
func (h *AdminHandler) GetOverview(c *fiber.Ctx) error {
	models := []struct{ name, table string }{
		{"user", `"user"`},
		{"topic", "topic"},
		{"topic_reply", "topic_reply"},
		{"topic_comment", "topic_comment"},
		{"galgame", "galgame"},
		{"galgame_resource", "galgame_resource"},
		{"galgame_comment", "galgame_comment"},
		{"message", "message"},
	}

	items := make([]overviewItem, len(models))
	for i, m := range models {
		var count int64
		h.db.Table(m.table).Count(&count)
		items[i] = overviewItem{Name: m.name, Label: m.name, Count: count}
	}

	return response.OK(c, items)
}

// GetStats returns daily counts for the last N days.
// GET /api/admin/overview/stats
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	var req struct {
		Days int `query:"days" validate:"min=1,max=365"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Days == 0 {
		req.Days = 30
	}

	since := time.Now().AddDate(0, 0, -req.Days)

	tables := []string{"user", "topic", "galgame"}
	type dailyStat struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	result := make(map[string][]dailyStat)
	for _, table := range tables {
		var stats []dailyStat
		quotedTable := table
		if table == "user" {
			quotedTable = `"user"`
		}
		h.db.Raw(fmt.Sprintf(`
			SELECT date_trunc('day', created)::date::text AS date, COUNT(*) AS count
			FROM %s WHERE created >= ? GROUP BY 1 ORDER BY 1
		`, quotedTable), since).Scan(&stats)
		result[table] = stats
	}

	return response.OK(c, result)
}

// ──────────────────────────────────────────
// Settings
// ──────────────────────────────────────────

// GetRegisterSetting returns whether registration is disabled.
// GET /api/admin/setting/register
func (h *AdminHandler) GetRegisterSetting(c *fiber.Ctx) error {
	val, err := h.rdb.Get(context.Background(), redisDisableRegisterKey).Result()
	disabled := err == nil && val == "1"
	return response.OK(c, fiber.Map{"registerStatus": !disabled})
}

// ToggleRegisterSetting toggles the registration on/off.
// PUT /api/admin/setting/register
func (h *AdminHandler) ToggleRegisterSetting(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	ctx := context.Background()
	val, _ := h.rdb.Get(ctx, redisDisableRegisterKey).Result()
	if val == "1" {
		h.rdb.Del(ctx, redisDisableRegisterKey)
	} else {
		h.rdb.Set(ctx, redisDisableRegisterKey, "1", 0)
	}

	return response.OKMessage(c, "注册设置已更新")
}

// ──────────────────────────────────────────
// User management
// ──────────────────────────────────────────

// GetUserList returns paginated user list for admin.
// GET /api/admin/user
func (h *AdminHandler) GetUserList(c *fiber.Ctx) error {
	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=100"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type userRow struct {
		ID      int       `json:"id"`
		Name    string    `json:"name"`
		Avatar  string    `json:"avatar"`
		Status  int       `json:"status"`
		Created time.Time `json:"created"`
	}

	var users []userRow
	var total int64

	h.db.Table(`"user"`).Count(&total)
	h.db.Table(`"user"`).
		Select("id, name, avatar, status, created").
		Order("created DESC").
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&users)

	return response.OK(c, fiber.Map{"users": users, "totalCount": total})
}

// SearchUsers searches users by name for admin.
// GET /api/admin/user/search
func (h *AdminHandler) SearchUsers(c *fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return response.Error(c, errors.ErrBadRequest("搜索关键词不能为空"))
	}

	type userRow struct {
		ID      int       `json:"id"`
		Name    string    `json:"name"`
		Avatar  string    `json:"avatar"`
		Status  int       `json:"status"`
		Created time.Time `json:"created"`
	}

	var users []userRow
	h.db.Table(`"user"`).
		Select("id, name, avatar, status, created").
		Where("name ILIKE ?", "%"+q+"%").
		Limit(50).
		Find(&users)

	return response.OK(c, users)
}
