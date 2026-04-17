package repository

import (
	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

// CommentRow is the joined comment+user row used for list/detail endpoints.
type CommentRow struct {
	ID         int     `gorm:"column:id"`
	Content    string  `gorm:"column:content"`
	ParentID   *int    `gorm:"column:parent_id"`
	UserID     int     `gorm:"column:user_id"`
	UserName   string  `gorm:"column:user_name"`
	UserAvatar string  `gorm:"column:user_avatar"`
	Created    string  `gorm:"column:created"`
	Edited     *string `gorm:"column:edited"`
}

// DetailCommentRow is the joined row used by the website detail endpoint.
type DetailCommentRow struct {
	ID         int    `gorm:"column:id"`
	Content    string `gorm:"column:content"`
	UserID     int    `gorm:"column:user_id"`
	UserName   string `gorm:"column:user_name"`
	UserAvatar string `gorm:"column:user_avatar"`
	Created    string `gorm:"column:created"`
	Updated    string `gorm:"column:updated"`
}

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// FindByWebsite returns all comments for a website joined with the author user.
func (r *CommentRepository) FindByWebsite(websiteID int) []CommentRow {
	var rows []CommentRow
	r.db.Table("galgame_website_comment c").
		Select(`c.id, c.content, c.parent_id, c.user_id,
			u.name AS user_name, u.avatar AS user_avatar,
			c.created, c.edited`).
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`).
		Where("c.website_id = ?", websiteID).
		Order("c.created DESC").
		Scan(&rows)
	return rows
}

// FindByWebsiteForDetail returns a slim comment+user projection used by the
// website detail endpoint.
func (r *CommentRepository) FindByWebsiteForDetail(websiteID int) []DetailCommentRow {
	var rows []DetailCommentRow
	r.db.Table("galgame_website_comment c").
		Select(`c.id, c.content, c.user_id,
			u.name AS user_name, u.avatar AS user_avatar,
			c.created, c.updated`).
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`).
		Where("c.website_id = ?", websiteID).
		Order("c.created DESC").
		Scan(&rows)
	return rows
}

// FindByID loads a single comment.
func (r *CommentRepository) FindByID(id int) (*model.GalgameWebsiteComment, error) {
	var comment model.GalgameWebsiteComment
	if err := r.db.First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new comment row.
func (r *CommentRepository) Create(comment *model.GalgameWebsiteComment) error {
	return r.db.Create(comment).Error
}

// Delete removes a comment by reference.
func (r *CommentRepository) Delete(comment *model.GalgameWebsiteComment) {
	r.db.Delete(comment)
}
