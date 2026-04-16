package common

import (
	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RankingHandler struct {
	db     *gorm.DB
	wikiGC *galgameClient.GalgameClient
}

func NewRankingHandler(db *gorm.DB, gc *galgameClient.GalgameClient) *RankingHandler {
	return &RankingHandler{db: db, wikiGC: gc}
}

// GetGalgameRanking returns galgame ranking by local interaction fields.
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

	// Step 1: Query local galgame table for IDs + sort value
	type localRow struct {
		ID    int `gorm:"column:id"`
		Value int `gorm:"column:value"`
	}
	var rows []localRow
	h.db.Table("galgame").
		Select("id, "+req.SortField+" AS value").
		Order(req.SortField+" "+req.SortOrder).
		Offset((req.Page-1)*req.Limit).
		Limit(req.Limit).
		Scan(&rows)

	if len(rows) == 0 {
		return response.OK(c, []fiber.Map{})
	}

	// Step 2: Batch fetch metadata from wiki
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	briefMap, appErr := h.wikiGC.GetBatch(c.Context(), ids)
	if appErr != nil {
		return response.OK(c, []fiber.Map{})
	}

	// Step 3: Batch fetch users
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	type userRow struct {
		ID     int    `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Avatar string `gorm:"column:avatar"`
	}
	var users []userRow
	if len(userIDs) > 0 {
		h.db.Table(`"user"`).Select("id, name, avatar").
			Where("id IN ?", userIDs).Scan(&users)
	}
	userMap := make(map[int]userRow, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Step 4: Assemble
	items := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		b, ok := briefMap[r.ID]
		if !ok {
			continue
		}
		u := userMap[b.UserID]
		items = append(items, fiber.Map{
			"id": r.ID,
			"name": fiber.Map{
				"en-us": b.NameEnUs, "ja-jp": b.NameJaJp,
				"zh-cn": b.NameZhCn, "zh-tw": b.NameZhTw,
			},
			"user":      fiber.Map{"id": u.ID, "name": u.Name, "avatar": u.Avatar},
			"banner":    b.Banner,
			"value":     r.Value,
			"sortField": req.SortField,
		})
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
		ID, UserID, Value          int
		Title, UserName, UserAvatar string
	}
	var rows []row
	h.db.Table("topic t").
		Select(`t.id, t.title, t.user_id, u.name AS user_name, u.avatar AS user_avatar,
			t.`+req.SortField+` AS value`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1").
		Order("t."+req.SortField+" "+req.SortOrder).
		Offset((req.Page-1)*req.Limit).Limit(req.Limit).
		Find(&rows)

	items := make([]fiber.Map, len(rows))
	for i, r := range rows {
		items[i] = fiber.Map{
			"id": r.ID, "title": r.Title,
			"user":      fiber.Map{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvatar},
			"value":     r.Value,
			"sortField": req.SortField,
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
		Order(req.SortField+" "+req.SortOrder).
		Offset((req.Page-1)*req.Limit).Limit(req.Limit).
		Find(&rows)

	return response.OK(c, rows)
}
