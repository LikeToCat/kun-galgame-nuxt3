package common

import (
	"time"

	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SectionHandler struct {
	db *gorm.DB
}

func NewSectionHandler(db *gorm.DB) *SectionHandler {
	return &SectionHandler{db: db}
}

// GetSectionTopics returns topics filtered by section.
// GET /api/section
func (h *SectionHandler) GetSectionTopics(c *fiber.Ctx) error {
	var req struct {
		Section   string `query:"section" validate:"required"`
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=30"`
		SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type topicRow struct {
		ID            int
		Title         string
		Content       string
		View          int
		LikeCount     int
		ReplyCount    int
		Status        int
		IsNSFW        bool
		BestAnswerID  *int
		UserID        int
		UserName      string
		UserAvatar    string
		Created       time.Time
	}

	var rows []topicRow
	var total int64

	query := h.db.Table("topic t").
		Select(`t.id, t.title, SUBSTRING(t.content, 1, 233) AS content,
			t.view, t.like_count, t.reply_count, t.status, t.is_nsfw,
			t.best_answer_id, t.user_id,
			u.name AS user_name, u.avatar AS user_avatar, t.created`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Joins("JOIN topic_section_relation tsr ON tsr.topic_id = t.id").
		Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
		Where("ts.name = ? AND t.status != 1", req.Section)

	query.Count(&total)

	query.Order("t.created " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(&rows)

	type item struct {
		ID           int       `json:"id"`
		Title        string    `json:"title"`
		Content      string    `json:"content"`
		View         int       `json:"view"`
		LikeCount    int       `json:"likeCount"`
		ReplyCount   int       `json:"replyCount"`
		HasBestAnswer bool     `json:"hasBestAnswer"`
		IsNSFW       bool      `json:"isNSFWTopic"`
		User         map[string]any `json:"user"`
		Created      time.Time `json:"created"`
	}

	items := make([]item, len(rows))
	for i, r := range rows {
		items[i] = item{
			ID: r.ID, Title: r.Title, Content: r.Content,
			View: r.View, LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			HasBestAnswer: r.BestAnswerID != nil, IsNSFW: r.IsNSFW,
			User:    map[string]any{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvatar},
			Created: r.Created,
		}
	}

	return response.Paginated(c, items, total)
}

// GetCategories returns topic category stats.
// GET /api/category
func (h *SectionHandler) GetCategories(c *fiber.Ctx) error {
	type catStat struct {
		Name       string `json:"name"`
		TopicCount int64  `json:"topicCount"`
		ViewCount  int64  `json:"viewCount"`
	}

	categories := []string{"galgame", "technique", "others"}
	stats := make([]catStat, len(categories))

	for i, cat := range categories {
		var topicCount, viewCount int64
		h.db.Table("topic").
			Where("category = ? AND status != 1", cat).
			Count(&topicCount)
		h.db.Table("topic").
			Select("COALESCE(SUM(view), 0)").
			Where("category = ? AND status != 1", cat).
			Scan(&viewCount)
		stats[i] = catStat{
			Name:       cat,
			TopicCount: topicCount,
			ViewCount:  viewCount,
		}
	}

	return response.OK(c, stats)
}
