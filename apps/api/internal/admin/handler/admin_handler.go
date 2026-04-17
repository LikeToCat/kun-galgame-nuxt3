package handler

import (
	"context"
	"fmt"
	"time"

	galgameClient "kun-galgame-api/internal/galgame/client"
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
	db     *gorm.DB
	rdb    *redis.Client
	wikiGC *galgameClient.GalgameClient
}

func NewAdminHandler(db *gorm.DB, rdb *redis.Client, gc *galgameClient.GalgameClient) *AdminHandler {
	return &AdminHandler{db: db, rdb: rdb, wikiGC: gc}
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
	models := []struct{ name, table, label string }{
		{"user", `"user"`, "用户"},
		{"topic", "topic", "话题"},
		{"topic_reply", "topic_reply", "话题回复"},
		{"topic_comment", "topic_comment", "话题评论"},
		{"galgame", "galgame", "Galgame"},
		{"galgame_resource", "galgame_resource", "Galgame 资源"},
		{"galgame_comment", "galgame_comment", "Galgame 评论"},
		{"galgame_website", "galgame_website", "Galgame 网站"},
		{"galgame_website_comment", "galgame_website_comment", "Galgame 网站评论"},
		{"chat_message", "chat_message", "聊天消息"},
	}

	items := make([]overviewItem, 0, len(models)+7)
	for _, m := range models {
		var count int64
		h.db.Table(m.table).Count(&count)
		items = append(items, overviewItem{Name: m.name, Label: m.label, Count: count})
	}

	// Merge wiki totals (non-blocking)
	wikiModels := []struct{ key, label string }{
		{"galgame_tag", "Galgame 标签"},
		{"galgame_official", "Galgame 会社"},
		{"galgame_engine", "Galgame 引擎"},
		{"galgame_series", "Galgame 系列"},
		{"galgame_link", "Galgame 链接"},
		{"galgame_pr", "Galgame PR"},
		{"galgame_revision", "Galgame 编辑历史"},
	}
	if wikiStats, err := h.wikiGC.GetAdminStats(c.Context(), 1); err == nil && wikiStats != nil {
		for _, m := range wikiModels {
			items = append(items, overviewItem{Name: m.key, Label: m.label, Count: wikiStats.Totals[m.key]})
		}
	} else {
		for _, m := range wikiModels {
			items = append(items, overviewItem{Name: m.key, Label: m.label, Count: 0})
		}
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

	// All tables the frontend expects
	tables := []struct{ key, table string }{
		{"user", `"user"`},
		{"topic", "topic"},
		{"topic_reply", "topic_reply"},
		{"topic_comment", "topic_comment"},
		{"galgame", "galgame"},
		{"galgame_resource", "galgame_resource"},
		{"galgame_comment", "galgame_comment"},
		{"galgame_website", "galgame_website"},
		{"galgame_website_comment", "galgame_website_comment"},
		{"chat_message", "chat_message"},
	}

	type dailyStat struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:count"`
	}

	// Collect per-table daily counts, indexed by date
	dateMap := make(map[string]map[string]int64) // date -> table -> count
	for _, t := range tables {
		var stats []dailyStat
		h.db.Raw(fmt.Sprintf(`
			SELECT date_trunc('day', created)::date::text AS date, COUNT(*) AS count
			FROM %s WHERE created >= ? GROUP BY 1 ORDER BY 1
		`, t.table), since).Scan(&stats)
		for _, s := range stats {
			if dateMap[s.Date] == nil {
				dateMap[s.Date] = make(map[string]int64)
			}
			dateMap[s.Date][t.key] = s.Count
		}
	}

	// Merge wiki daily stats (non-blocking)
	wikiKeys := []string{"galgame_tag", "galgame_official", "galgame_engine", "galgame_series", "galgame_link", "galgame_pr", "galgame_revision"}
	if wikiStats, err := h.wikiGC.GetAdminStats(c.Context(), req.Days); err == nil && wikiStats != nil {
		for _, day := range wikiStats.Daily {
			date, _ := day["date"].(string)
			if date == "" {
				continue
			}
			if dateMap[date] == nil {
				dateMap[date] = make(map[string]int64)
			}
			for _, key := range wikiKeys {
				if v, ok := day[key]; ok {
					switch n := v.(type) {
					case float64:
						dateMap[date][key] = int64(n)
					case int64:
						dateMap[date][key] = n
					}
				}
			}
		}
	}

	// Build sorted flat array: [{date, user, topic, ...}, ...]
	allKeys := make([]string, 0, len(tables)+len(wikiKeys))
	for _, t := range tables {
		allKeys = append(allKeys, t.key)
	}
	allKeys = append(allKeys, wikiKeys...)

	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sortStrings(dates)

	result := make([]fiber.Map, len(dates))
	for i, d := range dates {
		row := fiber.Map{"date": d}
		for _, key := range allKeys {
			row[key] = dateMap[d][key]
		}
		result[i] = row
	}

	return response.OK(c, result)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
