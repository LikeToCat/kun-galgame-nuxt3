package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) DB() *gorm.DB {
	return r.db
}

// CommentRow holds a galgame comment row. Identity is hydrated by the
// service layer via userclient.
type CommentRow struct {
	ID           int
	Content      string
	GalgameID    int
	UserID       int
	TargetUserID *int
	LikeCount    int
	CreatedAt    string
}

// CountByGalgame returns total comment count for a galgame.
func (r *CommentRepository) CountByGalgame(galgameID int) int64 {
	var total int64
	r.db.Model(&model.GalgameComment{}).
		Where("galgame_id = ?", galgameID).
		Count(&total)
	return total
}

// FindPaginated returns comment rows for a galgame, ordered by created DESC.
// Identity is hydrated at the service layer via userclient.
func (r *CommentRepository) FindPaginated(galgameID, page, limit int) []CommentRow {
	var rows []CommentRow
	r.db.Table("galgame_comment gc").
		Select(`gc.id, gc.content, gc.galgame_id, gc.user_id,
			gc.target_user_id, gc.like_count, gc.created AS created_at`).
		Where("gc.galgame_id = ?", galgameID).
		Order("gc.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return rows
}

// FindByID returns a comment by its primary key.
func (r *CommentRepository) FindByID(id int) (*model.GalgameComment, error) {
	var comment model.GalgameComment
	err := r.db.First(&comment, id).Error
	return &comment, err
}
