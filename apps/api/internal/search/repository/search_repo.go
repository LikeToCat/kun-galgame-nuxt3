package repository

import (
	"time"

	"gorm.io/gorm"
)

type SearchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

// TopicRow / ReplyRow / CommentRow no longer carry user identity; the service
// layer hydrates name/avatar via userclient since the user table is no longer
// the source of truth.
type TopicRow struct {
	ID               int
	Title            string
	View             int
	Status           int
	LikeCount        int
	ReplyCount       int
	CommentCount     int
	StatusUpdateTime time.Time
	UserID           int
}

type ReplyRow struct {
	ID         int
	TopicID    int
	TopicTitle string
	Content    string
	Floor      int
	UserID     int
	Created    time.Time
}

type CommentRow struct {
	ID         int
	TopicID    int
	TopicTitle string
	Content    string
	UserID     int
	Created    time.Time
}

// ──────────────────────────────────────────
// Queries
// ──────────────────────────────────────────

// SearchTopics fulltext-searches topics by title/content/category. Identity
// is hydrated by the service layer via userclient.
func (r *SearchRepository) SearchTopics(keywords []string, page, limit int) (rows []TopicRow, total int64) {
	query := r.db.Table("topic t").
		Select(`t.id, t.title, t.view, t.status, t.like_count, t.reply_count,
			t.comment_count, t.status_update_time, t.user_id`).
		Where("t.status != 1")
	for _, kw := range keywords {
		like := "%" + kw + "%"
		query = query.Where("(t.title ILIKE ? OR t.content ILIKE ? OR t.category ILIKE ?)",
			like, like, like)
	}

	query.Count(&total)
	query.Order("t.status_update_time DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}

// SearchReplies searches topic replies by content. Identity is hydrated by
// the service layer via userclient.
func (r *SearchRepository) SearchReplies(keywords []string, page, limit int) (rows []ReplyRow, total int64) {
	query := r.db.Table("topic_reply r").
		Select(`r.id, r.topic_id, t.title AS topic_title,
			SUBSTRING(r.content, 1, 233) AS content, r.floor,
			r.user_id, r.created`).
		Joins("LEFT JOIN topic t ON t.id = r.topic_id")
	for _, kw := range keywords {
		query = query.Where("r.content ILIKE ?", "%"+kw+"%")
	}

	query.Count(&total)
	query.Order("r.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}

// SearchComments searches topic comments by content. Identity is hydrated by
// the service layer via userclient.
func (r *SearchRepository) SearchComments(keywords []string, page, limit int) (rows []CommentRow, total int64) {
	query := r.db.Table("topic_comment c").
		Select(`c.id, c.topic_id, t.title AS topic_title,
			SUBSTRING(c.content, 1, 233) AS content,
			c.user_id, c.created`).
		Joins("LEFT JOIN topic t ON t.id = c.topic_id")
	for _, kw := range keywords {
		query = query.Where("c.content ILIKE ?", "%"+kw+"%")
	}

	query.Count(&total)
	query.Order("c.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}
