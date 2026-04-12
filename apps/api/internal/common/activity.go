package common

import (
	"fmt"
	"time"

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

type activityFetcher struct {
	table      string
	typeStr    string
	linkPrefix string
	contentCol string
	timeCol    string
	userIDCol  string
	extraWhere string
	linkCol    string
}

var fetchers = map[string]activityFetcher{
	"TOPIC_CREATION": {
		table: "topic", typeStr: "TOPIC_CREATION",
		linkPrefix: "/topic/", contentCol: "title", timeCol: "created",
		userIDCol: "user_id", extraWhere: "status != 1",
	},
	"TOPIC_REPLY_CREATION": {
		table: "topic_reply", typeStr: "TOPIC_REPLY_CREATION",
		linkPrefix: "/topic/", contentCol: "SUBSTRING(content, 1, 100)", timeCol: "created",
		userIDCol: "user_id", linkCol: "topic_id",
	},
	"TOPIC_COMMENT_CREATION": {
		table: "topic_comment", typeStr: "TOPIC_COMMENT_CREATION",
		linkPrefix: "/topic/", contentCol: "SUBSTRING(content, 1, 100)", timeCol: "created",
		userIDCol: "user_id", linkCol: "topic_id",
	},
	"GALGAME_COMMENT_CREATION": {
		table: "galgame_comment", typeStr: "GALGAME_COMMENT_CREATION",
		linkPrefix: "/galgame/", contentCol: "SUBSTRING(content, 1, 100)", timeCol: "created",
		userIDCol: "user_id", linkCol: "galgame_id",
	},
	"TOOLSET_CREATION": {
		table: "galgame_toolset", typeStr: "TOOLSET_CREATION",
		linkPrefix: "/toolset/", contentCol: "name", timeCol: "created",
		userIDCol: "user_id", extraWhere: "status != 1",
	},
	"TOOLSET_COMMENT_CREATION": {
		table: "galgame_toolset_comment", typeStr: "TOOLSET_COMMENT_CREATION",
		linkPrefix: "/toolset/", contentCol: "SUBSTRING(content, 1, 100)", timeCol: "created",
		userIDCol: "user_id", linkCol: "toolset_id",
	},
	"TODO_CREATION": {
		table: "todo", typeStr: "TODO_CREATION",
		linkPrefix: "/update", contentCol: "content_zh_cn", timeCol: "created",
		userIDCol: "user_id",
	},
	"UPDATE_LOG_CREATION": {
		table: "update_log", typeStr: "UPDATE_LOG_CREATION",
		linkPrefix: "/update", contentCol: "content_zh_cn", timeCol: "created",
		userIDCol: "user_id",
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

	fetcher, ok := fetchers[req.Type]
	if !ok {
		return response.Paginated(c, []activityItem{}, 0)
	}

	items, total := h.fetchActivity(fetcher, req.Page, req.Limit)
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

func (h *ActivityHandler) getTimeline(c *fiber.Ctx, page, limit int) error {
	// Fetch from all types, merge and sort by time
	var allItems []activityItem
	for _, f := range fetchers {
		items, _ := h.fetchActivity(f, 1, 50)
		allItems = append(allItems, items...)
	}

	// Sort by timestamp descending
	for i := 0; i < len(allItems); i++ {
		for j := i + 1; j < len(allItems); j++ {
			if allItems[j].Timestamp.After(allItems[i].Timestamp) {
				allItems[i], allItems[j] = allItems[j], allItems[i]
			}
		}
	}

	total := int64(len(allItems))
	start := (page - 1) * limit
	if start >= len(allItems) {
		return response.Paginated(c, []activityItem{}, total)
	}
	end := start + limit
	if end > len(allItems) {
		end = len(allItems)
	}

	return response.Paginated(c, allItems[start:end], total)
}

func (h *ActivityHandler) fetchActivity(f activityFetcher, page, limit int) ([]activityItem, int64) {
	type row struct {
		ID         int
		Content    string
		LinkID     int
		Timestamp  time.Time
		UserID     int
		UserName   string
		UserAvatar string
	}

	linkIDSelect := "t.id AS link_id"
	if f.linkCol != "" {
		linkIDSelect = fmt.Sprintf("t.%s AS link_id", f.linkCol)
	}

	where := "1=1"
	if f.extraWhere != "" {
		where = "t." + f.extraWhere
	}

	var rows []row
	var total int64

	countQuery := h.db.Table(fmt.Sprintf("%s t", f.table)).Where(where)
	countQuery.Count(&total)

	h.db.Table(fmt.Sprintf("%s t", f.table)).
		Select(fmt.Sprintf(`t.id, %s AS content, %s, t.%s AS timestamp,
			t.%s AS user_id, u.name AS user_name, u.avatar AS user_avatar`,
			f.contentCol, linkIDSelect, f.timeCol, f.userIDCol)).
		Joins(fmt.Sprintf(`LEFT JOIN "user" u ON u.id = t.%s`, f.userIDCol)).
		Where(where).
		Order(fmt.Sprintf("t.%s DESC", f.timeCol)).
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	items := make([]activityItem, len(rows))
	for i, r := range rows {
		linkID := r.LinkID
		if linkID == 0 {
			linkID = r.ID
		}
		items[i] = activityItem{
			UniqueID:  fmt.Sprintf("%s-%d", f.typeStr, r.ID),
			Type:      f.typeStr,
			Timestamp: r.Timestamp,
			Link:      fmt.Sprintf("%s%d", f.linkPrefix, linkID),
			Content:   r.Content,
		}
		items[i].Actor.ID = r.UserID
		items[i].Actor.Name = r.UserName
		items[i].Actor.Avatar = r.UserAvatar
	}
	return items, total
}
