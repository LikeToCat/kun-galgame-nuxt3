package common

import (
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RankingHandler struct {
	db *gorm.DB
}

func NewRankingHandler(db *gorm.DB) *RankingHandler {
	return &RankingHandler{db: db}
}

type rankingItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Banner string `json:"banner,omitempty"`
	Title  string `json:"title,omitempty"`
	Bio    string `json:"bio,omitempty"`
	Value  int    `json:"value"`
	Field  string `json:"sortField"`
}

// GetGalgameRanking returns galgame ranking by various fields.
// GET /api/ranking/galgame
func (h *RankingHandler) GetGalgameRanking(c *fiber.Ctx) error {
	var req struct {
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=50"`
		SortField string `query:"sortField" validate:"required,oneof=view like_count favorite_count resource_count"`
		SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type row struct {
		ID      int    `json:"id"`
		NameEnUS string `json:"name_en_us"`
		NameJaJP string `json:"name_ja_jp"`
		NameZhCN string `json:"name_zh_cn"`
		NameZhTW string `json:"name_zh_tw"`
		Banner  string `json:"banner"`
		UserID  int
		UserName   string
		UserAvatar string
		Value   int    `json:"value"`
	}

	var rows []row
	h.db.Table("galgame g").
		Select(`g.id, g.name_en_us, g.name_ja_jp, g.name_zh_cn, g.name_zh_tw,
			g.banner, g.user_id, u.name AS user_name, u.avatar AS user_avatar,
			g.`+req.SortField+` AS value`).
		Joins(`LEFT JOIN "user" u ON u.id = g.user_id`).
		Where("g.status != 1").
		Order("g." + req.SortField + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&rows)

	type resultItem struct {
		ID     int               `json:"id"`
		Name   map[string]string `json:"name"`
		User   map[string]any    `json:"user"`
		Banner string            `json:"banner"`
		Value  int               `json:"value"`
		Field  string            `json:"sortField"`
	}

	items := make([]resultItem, len(rows))
	for i, r := range rows {
		items[i] = resultItem{
			ID: r.ID,
			Name: map[string]string{
				"en-us": r.NameEnUS, "ja-jp": r.NameJaJP,
				"zh-cn": r.NameZhCN, "zh-tw": r.NameZhTW,
			},
			User:   map[string]any{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvatar},
			Banner: r.Banner,
			Value:  r.Value,
			Field:  req.SortField,
		}
	}

	return response.OK(c, items)
}

// GetTopicRanking returns topic ranking.
// GET /api/ranking/topic
func (h *RankingHandler) GetTopicRanking(c *fiber.Ctx) error {
	var req struct {
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=50"`
		SortField string `query:"sortField" validate:"required,oneof=view upvote_count like_count reply_count comment_count favorite_count"`
		SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type row struct {
		ID         int
		Title      string
		UserID     int
		UserName   string
		UserAvatar string
		Value      int
	}

	var rows []row
	h.db.Table("topic t").
		Select(`t.id, t.title, t.user_id, u.name AS user_name, u.avatar AS user_avatar,
			t.`+req.SortField+` AS value`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1").
		Order("t." + req.SortField + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&rows)

	type resultItem struct {
		ID    int            `json:"id"`
		Title string         `json:"title"`
		User  map[string]any `json:"user"`
		Value int            `json:"value"`
		Field string         `json:"sortField"`
	}

	items := make([]resultItem, len(rows))
	for i, r := range rows {
		items[i] = resultItem{
			ID: r.ID, Title: r.Title,
			User:  map[string]any{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvatar},
			Value: r.Value, Field: req.SortField,
		}
	}

	return response.OK(c, items)
}

// GetUserRanking returns user ranking.
// GET /api/ranking/user
func (h *RankingHandler) GetUserRanking(c *fiber.Ctx) error {
	var req struct {
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=50"`
		SortField string `query:"sortField" validate:"required,oneof=moemoepoint"`
		SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type row struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
		Bio    string `json:"bio"`
		Value  int    `json:"value"`
	}

	var rows []row
	h.db.Table(`"user"`).
		Select(`id, name, avatar, bio, `+req.SortField+` AS value`).
		Where("status != 1").
		Order(req.SortField + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&rows)

	return response.OK(c, rows)
}
