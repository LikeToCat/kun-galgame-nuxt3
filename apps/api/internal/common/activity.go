package common

import (
	"fmt"
	"strings"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ActivityHandler struct {
	db *gorm.DB
}

func NewActivityHandler(db *gorm.DB) *ActivityHandler {
	return &ActivityHandler{db: db}
}

type activityItem struct {
	UniqueID  string    `json:"uniqueId"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	} `json:"actor"`
	Link    string `json:"link"`
	Content string `json:"content"`
}

// activitySource defines a single SQL sub-query that produces
// activity rows. The query MUST SELECT these exact columns:
//
//	type_str, id, content, link, created, user_id
//
// All column references must be fully qualified (t.xxx) to avoid
// ambiguity when later JOINed with the "user" table.
type activitySource struct {
	typeStr string
	query   string
}

var sources = map[string]activitySource{
	"TOPIC_CREATION": {
		typeStr: "TOPIC_CREATION",
		query: `SELECT 'TOPIC_CREATION' AS type_str, t.id,
			t.title AS content,
			'/topic/' || t.id AS link, t.created, t.user_id
			FROM topic t WHERE t.status != 1`,
	},
	"TOPIC_REPLY_CREATION": {
		typeStr: "TOPIC_REPLY_CREATION",
		query: `SELECT 'TOPIC_REPLY_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/topic/' || t.topic_id AS link, t.created, t.user_id
			FROM topic_reply t`,
	},
	"TOPIC_COMMENT_CREATION": {
		typeStr: "TOPIC_COMMENT_CREATION",
		query: `SELECT 'TOPIC_COMMENT_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/topic/' || t.topic_id AS link, t.created, t.user_id
			FROM topic_comment t`,
	},
	"GALGAME_CREATION": {
		typeStr: "GALGAME_CREATION",
		query: `SELECT 'GALGAME_CREATION' AS type_str, t.id,
			COALESCE(NULLIF(t.name_zh_cn,''), NULLIF(t.name_ja_jp,''),
				NULLIF(t.name_en_us,''), t.name_zh_tw) AS content,
			'/galgame/' || t.id AS link, t.created, t.user_id
			FROM galgame t WHERE t.status != 1`,
	},
	"GALGAME_COMMENT_CREATION": {
		typeStr: "GALGAME_COMMENT_CREATION",
		query: `SELECT 'GALGAME_COMMENT_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/galgame/' || t.galgame_id AS link, t.created, t.user_id
			FROM galgame_comment t`,
	},
	"GALGAME_RESOURCE_CREATION": {
		typeStr: "GALGAME_RESOURCE_CREATION",
		query: `SELECT 'GALGAME_RESOURCE_CREATION' AS type_str, t.id,
			COALESCE(NULLIF(t.note,''), t.type) AS content,
			'/galgame/' || t.galgame_id AS link, t.created, t.user_id
			FROM galgame_resource t`,
	},
	"GALGAME_RATING_CREATION": {
		typeStr: "GALGAME_RATING_CREATION",
		query: `SELECT 'GALGAME_RATING_CREATION' AS type_str, t.id,
			SUBSTRING(COALESCE(t.short_summary,''), 1, 100) AS content,
			'/galgame/' || t.galgame_id AS link, t.created, t.user_id
			FROM galgame_rating t`,
	},
	"GALGAME_RATING_COMMENT_CREATION": {
		typeStr: "GALGAME_RATING_COMMENT_CREATION",
		query: `SELECT 'GALGAME_RATING_COMMENT_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/galgame/' || t.galgame_rating_id AS link, t.created, t.user_id
			FROM galgame_rating_comment t`,
	},
	"GALGAME_PR_CREATION": {
		typeStr: "GALGAME_PR_CREATION",
		query: `SELECT 'GALGAME_PR_CREATION' AS type_str, t.id,
			COALESCE(NULLIF(t.note,''), 'PR') AS content,
			'/galgame/' || t.galgame_id AS link, t.created, t.user_id
			FROM galgame_pr t`,
	},
	"GALGAME_WEBSITE_CREATION": {
		typeStr: "GALGAME_WEBSITE_CREATION",
		query: `SELECT 'GALGAME_WEBSITE_CREATION' AS type_str, t.id,
			t.name AS content,
			'/website/' || t.id AS link, t.created, t.user_id
			FROM galgame_website t`,
	},
	"GALGAME_WEBSITE_COMMENT_CREATION": {
		typeStr: "GALGAME_WEBSITE_COMMENT_CREATION",
		query: `SELECT 'GALGAME_WEBSITE_COMMENT_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/website/' || t.website_id AS link, t.created, t.user_id
			FROM galgame_website_comment t`,
	},
	"TOOLSET_CREATION": {
		typeStr: "TOOLSET_CREATION",
		query: `SELECT 'TOOLSET_CREATION' AS type_str, t.id,
			t.name AS content,
			'/toolset/' || t.id AS link, t.created, t.user_id
			FROM galgame_toolset t WHERE t.status != 1`,
	},
	"TOOLSET_RESOURCE_CREATION": {
		typeStr: "TOOLSET_RESOURCE_CREATION",
		query: `SELECT 'TOOLSET_RESOURCE_CREATION' AS type_str, t.id,
			COALESCE(NULLIF(t.note,''), t.content) AS content,
			'/toolset/' || t.toolset_id AS link, t.created, t.user_id
			FROM galgame_toolset_resource t`,
	},
	"TOOLSET_COMMENT_CREATION": {
		typeStr: "TOOLSET_COMMENT_CREATION",
		query: `SELECT 'TOOLSET_COMMENT_CREATION' AS type_str, t.id,
			SUBSTRING(t.content, 1, 100) AS content,
			'/toolset/' || t.toolset_id AS link, t.created, t.user_id
			FROM galgame_toolset_comment t`,
	},
	"TODO_CREATION": {
		typeStr: "TODO_CREATION",
		query: `SELECT 'TODO_CREATION' AS type_str, t.id,
			t.content_zh_cn AS content,
			'/update' AS link, t.created, t.user_id
			FROM todo t`,
	},
	"UPDATE_LOG_CREATION": {
		typeStr: "UPDATE_LOG_CREATION",
		query: `SELECT 'UPDATE_LOG_CREATION' AS type_str, t.id,
			t.content_zh_cn AS content,
			'/update' AS link, t.created, t.user_id
			FROM update_log t`,
	},
	"MESSAGE_UPVOTE": {
		typeStr: "MESSAGE_UPVOTE",
		query: `SELECT 'MESSAGE_UPVOTE' AS type_str, t.id, t.content,
			t.link, t.created, t.sender_id AS user_id
			FROM message t WHERE t.type = 'upvoted'`,
	},
	"MESSAGE_SOLUTION": {
		typeStr: "MESSAGE_SOLUTION",
		query: `SELECT 'MESSAGE_SOLUTION' AS type_str, t.id, t.content,
			t.link, t.created, t.sender_id AS user_id
			FROM message t WHERE t.type = 'solution'`,
	},
}

// GetActivity returns activity feed filtered by type.
// GET /api/activity
func (h *ActivityHandler) GetActivity(c *fiber.Ctx) error {
	var req struct {
		Page  int    `query:"page" validate:"min=1"`
		Limit int    `query:"limit" validate:"min=1,max=50"`
		Type  string `query:"type" validate:"required"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if req.Type == "all" {
		return h.getTimeline(c, req.Page, req.Limit)
	}

	src, ok := sources[req.Type]
	if !ok {
		return response.Paginated(c, []activityItem{}, 0)
	}

	items, total, err := h.fetch(src, req.Page, req.Limit)
	if err != nil {
		return response.Error(c, errors.ErrInternal("查询活动数据失败"))
	}
	return response.Paginated(c, items, total)
}

// GetTimeline returns mixed activity timeline.
// GET /api/activity/timeline
func (h *ActivityHandler) GetTimeline(c *fiber.Ctx) error {
	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return h.getTimeline(c, req.Page, req.Limit)
}

// getTimeline uses a single UNION ALL query across all source tables,
// letting PostgreSQL handle the sort and pagination in one pass.
func (h *ActivityHandler) getTimeline(
	c *fiber.Ctx, page, limit int,
) error {
	union := buildUnionAll()

	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) AS u`, union)
	if err := h.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return response.Error(c, errors.ErrInternal("查询活动总数失败"))
	}

	type row struct {
		TypeStr  string    `gorm:"column:type_str"`
		ID       int       `gorm:"column:id"`
		Content  string    `gorm:"column:content"`
		Link     string    `gorm:"column:link"`
		Created  time.Time `gorm:"column:created"`
		UserID   int       `gorm:"column:user_id"`
		UserName string    `gorm:"column:user_name"`
		Avatar   string    `gorm:"column:avatar"`
	}

	dataSQL := fmt.Sprintf(
		`SELECT u.*, usr.name AS user_name, usr.avatar
		FROM (%s) AS u
		LEFT JOIN "user" usr ON usr.id = u.user_id
		ORDER BY u.created DESC
		LIMIT %d OFFSET %d`,
		union, limit, (page-1)*limit,
	)
	var rows []row
	if err := h.db.Raw(dataSQL).Scan(&rows).Error; err != nil {
		return response.Error(c, errors.ErrInternal("查询活动列表失败"))
	}

	items := make([]activityItem, len(rows))
	for i, r := range rows {
		items[i] = activityItem{
			UniqueID:  fmt.Sprintf("%s-%d", r.TypeStr, r.ID),
			Type:      r.TypeStr,
			Content:   r.Content,
			Link:      r.Link,
			Timestamp: r.Created,
		}
		items[i].Actor.ID = r.UserID
		items[i].Actor.Name = r.UserName
		items[i].Actor.Avatar = r.Avatar
	}

	return response.Paginated(c, items, total)
}

// buildUnionAll joins all source queries with UNION ALL.
func buildUnionAll() string {
	parts := make([]string, 0, len(sources))
	for _, src := range sources {
		parts = append(parts, "("+src.query+")")
	}
	return strings.Join(parts, " UNION ALL ")
}

// fetch runs a single source query with user JOIN, pagination,
// and count. Returns error on DB failure instead of silently
// returning empty results.
func (h *ActivityHandler) fetch(
	src activitySource, page, limit int,
) ([]activityItem, int64, error) {
	type row struct {
		TypeStr  string    `gorm:"column:type_str"`
		ID       int       `gorm:"column:id"`
		Content  string    `gorm:"column:content"`
		Link     string    `gorm:"column:link"`
		Created  time.Time `gorm:"column:created"`
		UserID   int       `gorm:"column:user_id"`
		UserName string    `gorm:"column:user_name"`
		Avatar   string    `gorm:"column:avatar"`
	}

	countSQL := fmt.Sprintf(
		`SELECT COUNT(*) FROM (%s) AS sub`, src.query,
	)
	var total int64
	if err := h.db.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	dataSQL := fmt.Sprintf(
		`SELECT sub.*, u.name AS user_name, u.avatar
		FROM (%s) AS sub
		LEFT JOIN "user" u ON u.id = sub.user_id
		ORDER BY sub.created DESC
		LIMIT %d OFFSET %d`,
		src.query, limit, (page-1)*limit,
	)
	var rows []row
	if err := h.db.Raw(dataSQL).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]activityItem, len(rows))
	for i, r := range rows {
		items[i] = activityItem{
			UniqueID:  fmt.Sprintf("%s-%d", r.TypeStr, r.ID),
			Type:      r.TypeStr,
			Content:   r.Content,
			Link:      r.Link,
			Timestamp: r.Created,
		}
		items[i].Actor.ID = r.UserID
		items[i].Actor.Name = r.UserName
		items[i].Actor.Avatar = r.Avatar
	}
	return items, total, nil
}
