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
	UserName         string
	UserAvatar       string
}

type UserRow struct {
	ID          int       `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Avatar      string    `gorm:"column:avatar"`
	Bio         string    `gorm:"column:bio"`
	Moemoepoint int       `gorm:"column:moemoepoint"`
	Created     time.Time `gorm:"column:created"`
}

type ReplyRow struct {
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

type CommentRow struct {
	ID         int
	TopicID    int
	TopicTitle string
	Content    string
	UserID     int
	UserName   string
	UserAvatar string
	Created    time.Time
}

// ──────────────────────────────────────────
// Queries
// ──────────────────────────────────────────

// SearchTopics fulltext-searches topics by title/content/category.
func (r *SearchRepository) SearchTopics(keywords []string, page, limit int) (rows []TopicRow, total int64) {
	query := r.db.Table("topic t").
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

	query.Count(&total)
	query.Order("t.status_update_time DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}

// SearchUsers searches users by name.
func (r *SearchRepository) SearchUsers(keywords []string, page, limit int) (rows []UserRow, total int64) {
	query := r.db.Table(`"user"`).
		Select("id, name, avatar, bio, moemoepoint, created").
		Where("status != 1")
	for _, kw := range keywords {
		query = query.Where("name ILIKE ?", "%"+kw+"%")
	}

	query.Count(&total)
	query.Order("moemoepoint DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}

// SearchReplies searches topic replies by content.
func (r *SearchRepository) SearchReplies(keywords []string, page, limit int) (rows []ReplyRow, total int64) {
	query := r.db.Table("topic_reply r").
		Select(`r.id, r.topic_id, t.title AS topic_title,
			SUBSTRING(r.content, 1, 233) AS content, r.floor,
			r.user_id, u.name AS user_name, u.avatar AS user_avatar, r.created`).
		Joins("LEFT JOIN topic t ON t.id = r.topic_id").
		Joins(`LEFT JOIN "user" u ON u.id = r.user_id`)
	for _, kw := range keywords {
		query = query.Where("r.content ILIKE ?", "%"+kw+"%")
	}

	query.Count(&total)
	query.Order("r.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}

// SearchComments searches topic comments by content.
func (r *SearchRepository) SearchComments(keywords []string, page, limit int) (rows []CommentRow, total int64) {
	query := r.db.Table("topic_comment c").
		Select(`c.id, c.topic_id, t.title AS topic_title,
			SUBSTRING(c.content, 1, 233) AS content,
			c.user_id, u.name AS user_name, u.avatar AS user_avatar, c.created`).
		Joins("LEFT JOIN topic t ON t.id = c.topic_id").
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`)
	for _, kw := range keywords {
		query = query.Where("c.content ILIKE ?", "%"+kw+"%")
	}

	query.Count(&total)
	query.Order("c.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return
}
