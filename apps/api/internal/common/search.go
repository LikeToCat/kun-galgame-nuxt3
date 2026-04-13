package common

import (
	"strings"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

// Search performs keyword search across different types.
// GET /api/search
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	var req struct {
		Keywords string `query:"keywords" validate:"required,max=107"`
		Type     string `query:"type" validate:"required,oneof=topic galgame user reply comment"`
		Page     int    `query:"page" validate:"min=1"`
		Limit    int    `query:"limit" validate:"min=1,max=12"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	keywords := strings.Fields(strings.TrimSpace(req.Keywords))
	if len(keywords) == 0 {
		return response.Error(c, errors.ErrBadRequest("搜索关键词不能为空"))
	}

	switch req.Type {
	case "topic":
		return h.searchTopic(c, keywords, req.Page, req.Limit)
	case "user":
		return h.searchUser(c, keywords, req.Page, req.Limit)
	case "reply":
		return h.searchReply(c, keywords, req.Page, req.Limit)
	case "comment":
		return h.searchComment(c, keywords, req.Page, req.Limit)
	default:
		return response.OK(c, []any{})
	}
}

func (h *SearchHandler) searchTopic(c *fiber.Ctx, keywords []string, page, limit int) error {
	type dbRow struct {
		ID               int
		Title            string
		View             int
		Status           int
		LikeCount        int
		ReplyCount       int
		CommentCount     int
		StatusUpdateTime time.Time
		UserID           int
		UserName         string
		UserAvatar       string
	}

	query := h.db.Table("topic t").
		Select(`t.id, t.title, t.view, t.status, t.like_count, t.reply_count,
			t.comment_count, t.status_update_time,
			t.user_id, u.name AS user_name, u.avatar AS user_avatar`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1")

	for _, kw := range keywords {
		like := "%" + kw + "%"
		query = query.Where("(t.title ILIKE ? OR t.content ILIKE ? OR t.category ILIKE ?)",
			like, like, like)
	}

	var total int64
	query.Count(&total)

	var rows []dbRow
	query.Order("t.status_update_time DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	type userObj struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type item struct {
		ID               int       `json:"id"`
		Title            string    `json:"title"`
		View             int       `json:"view"`
		Status           int       `json:"status"`
		LikeCount        int       `json:"likeCount"`
		ReplyCount       int       `json:"replyCount"`
		CommentCount     int       `json:"commentCount"`
		StatusUpdateTime time.Time `json:"statusUpdateTime"`
		User             userObj   `json:"user"`
	}

	items := make([]item, len(rows))
	for i, r := range rows {
		items[i] = item{
			ID: r.ID, Title: r.Title, View: r.View, Status: r.Status,
			LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			CommentCount: r.CommentCount, StatusUpdateTime: r.StatusUpdateTime,
			User: userObj{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
		}
	}

	return response.Paginated(c, items, total)
}

func (h *SearchHandler) searchUser(c *fiber.Ctx, keywords []string, page, limit int) error {
	type row struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Avatar      string    `json:"avatar"`
		Bio         string    `json:"bio"`
		Moemoepoint int       `json:"moemoepoint"`
		Created     time.Time `json:"created"`
	}

	query := h.db.Table(`"user"`).
		Select("id, name, avatar, bio, moemoepoint, created").
		Where("status != 1")

	for _, kw := range keywords {
		query = query.Where("name ILIKE ?", "%"+kw+"%")
	}

	var total int64
	query.Count(&total)

	var rows []row
	query.Order("moemoepoint DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	return response.Paginated(c, rows, total)
}

func (h *SearchHandler) searchReply(c *fiber.Ctx, keywords []string, page, limit int) error {
	type dbRow struct {
		ID         int
		TopicID    int
		TopicTitle string
		Content    string
		Floor      int
		UserID     int
		UserName   string
		UserAvatar string
		Created    time.Time
	}

	query := h.db.Table("topic_reply r").
		Select(`r.id, r.topic_id, t.title AS topic_title,
			SUBSTRING(r.content, 1, 233) AS content, r.floor,
			r.user_id, u.name AS user_name, u.avatar AS user_avatar, r.created`).
		Joins("LEFT JOIN topic t ON t.id = r.topic_id").
		Joins(`LEFT JOIN "user" u ON u.id = r.user_id`)

	for _, kw := range keywords {
		query = query.Where("r.content ILIKE ?", "%"+kw+"%")
	}

	var total int64
	query.Count(&total)

	var rows []dbRow
	query.Order("r.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	type userObj struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type item struct {
		ID         int       `json:"id"`
		TopicID    int       `json:"topicId"`
		TopicTitle string    `json:"topicTitle"`
		Content    string    `json:"content"`
		Floor      int       `json:"floor"`
		User       userObj   `json:"user"`
		Created    time.Time `json:"created"`
	}

	items := make([]item, len(rows))
	for i, r := range rows {
		items[i] = item{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content, Floor: r.Floor,
			User:    userObj{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Created: r.Created,
		}
	}

	return response.Paginated(c, items, total)
}

func (h *SearchHandler) searchComment(c *fiber.Ctx, keywords []string, page, limit int) error {
	type dbRow struct {
		ID         int
		TopicID    int
		TopicTitle string
		Content    string
		UserID     int
		UserName   string
		UserAvatar string
		Created    time.Time
	}

	query := h.db.Table("topic_comment c").
		Select(`c.id, c.topic_id, t.title AS topic_title,
			SUBSTRING(c.content, 1, 233) AS content,
			c.user_id, u.name AS user_name, u.avatar AS user_avatar, c.created`).
		Joins("LEFT JOIN topic t ON t.id = c.topic_id").
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`)

	for _, kw := range keywords {
		query = query.Where("c.content ILIKE ?", "%"+kw+"%")
	}

	var total int64
	query.Count(&total)

	var rows []dbRow
	query.Order("c.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)

	type userObj struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type item struct {
		ID         int       `json:"id"`
		TopicID    int       `json:"topicId"`
		TopicTitle string    `json:"topicTitle"`
		Content    string    `json:"content"`
		User       userObj   `json:"user"`
		Created    time.Time `json:"created"`
	}

	items := make([]item, len(rows))
	for i, r := range rows {
		items[i] = item{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content,
			User:    userObj{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Created: r.Created,
		}
	}

	return response.Paginated(c, items, total)
}
