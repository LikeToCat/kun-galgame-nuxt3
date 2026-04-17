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

// CommentRow holds a joined comment row with user info.
type CommentRow struct {
	ID               int
	Content          string
	GalgameID        int
	UserID           int
	UserName         string
	UserAvatar       string
	TargetUserID     *int
	TargetUserName   string
	TargetUserAvatar string
	LikeCount        int
	CreatedAt        string
}

// CountByGalgame returns total comment count for a galgame.
func (r *CommentRepository) CountByGalgame(galgameID int) int64 {
	var total int64
	r.db.Model(&model.GalgameComment{}).
		Where("galgame_id = ?", galgameID).
		Count(&total)
	return total
}

// FindPaginated returns joined comment rows for a galgame, ordered by created DESC.
func (r *CommentRepository) FindPaginated(galgameID, page, limit int) []CommentRow {
	var rows []CommentRow
	r.db.Table("galgame_comment gc").
		Select(`gc.id, gc.content, gc.galgame_id, gc.user_id,
			u1.name AS user_name, u1.avatar AS user_avatar,
			gc.target_user_id, u2.name AS target_user_name, u2.avatar AS target_user_avatar,
			gc.like_count, gc.created AS created_at`).
		Joins(`LEFT JOIN "user" u1 ON u1.id = gc.user_id`).
		Joins(`LEFT JOIN "user" u2 ON u2.id = gc.target_user_id`).
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

// GetUserInfo fetches name and avatar for a user ID.
func (r *CommentRepository) GetUserInfo(userID int) (name, avatar string) {
	r.db.Table(`"user"`).
		Select("name, avatar").
		Where("id = ?", userID).
		Row().Scan(&name, &avatar)
	return
}
