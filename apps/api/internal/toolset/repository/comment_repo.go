package repository

import (
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"

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
// Reads
// ──────────────────────────────────────────

// CountByToolset returns total comment count for a toolset.
func (r *CommentRepository) CountByToolset(toolsetID int) int64 {
	var total int64
	r.db.Model(&model.GalgameToolsetComment{}).
		Where("toolset_id = ?", toolsetID).
		Count(&total)
	return total
}

// CountsForToolsets returns a map[toolsetID]count for a batch.
func (r *CommentRepository) CountsForToolsets(toolsetIDs []int) map[int]int {
	if len(toolsetIDs) == 0 {
		return map[int]int{}
	}
	type row struct {
		ToolsetID int
		Count     int
	}
	var rows []row
	r.db.Model(&model.GalgameToolsetComment{}).
		Select("toolset_id, COUNT(*) AS count").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, row := range rows {
		out[row.ToolsetID] = row.Count
	}
	return out
}

// FindPaginated returns the paginated comments for a toolset ordered by
// created in the requested direction. `sortOrder` accepts "asc" / "desc";
// any other value falls back to "desc".
func (r *CommentRepository) FindPaginated(toolsetID, page, limit int, sortOrder string) []model.GalgameToolsetComment {
	var comments []model.GalgameToolsetComment
	offset := (page - 1) * limit
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}
	r.db.Where("toolset_id = ?", toolsetID).
		Order("created " + dir).
		Offset(offset).Limit(limit).
		Find(&comments)
	return comments
}

// FindLatest returns the N most recent comments for a toolset.
func (r *CommentRepository) FindLatest(toolsetID, limit int) []model.GalgameToolsetComment {
	var comments []model.GalgameToolsetComment
	r.db.Where("toolset_id = ?", toolsetID).
		Order("created DESC").
		Limit(limit).
		Find(&comments)
	return comments
}

// FindByID loads a single comment.
func (r *CommentRepository) FindByID(id int) (*model.GalgameToolsetComment, error) {
	var comment model.GalgameToolsetComment
	if err := r.db.First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// FindUser returns a single UserBrief row (empty if not found).
func (r *CommentRepository) FindUser(userID int) userModel.UserBrief {
	var u userModel.UserBrief
	r.db.Where("id = ?", userID).First(&u)
	return u
}

// FindUsersByIDs batch-loads UserBriefs keyed by id.
func (r *CommentRepository) FindUsersByIDs(ids []int) map[int]userModel.UserBrief {
	if len(ids) == 0 {
		return map[int]userModel.UserBrief{}
	}
	var users []userModel.UserBrief
	r.db.Where("id IN ?", ids).Find(&users)
	out := make(map[int]userModel.UserBrief, len(users))
	for _, u := range users {
		out[u.ID] = u
	}
	return out
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new comment and returns the created row.
func (r *CommentRepository) Create(comment *model.GalgameToolsetComment) error {
	return r.db.Create(comment).Error
}

// UpdateContent sets the content and `edited` timestamp on a comment.
func (r *CommentRepository) UpdateContent(comment *model.GalgameToolsetComment, content string, editedAt time.Time) {
	r.db.Model(comment).Updates(map[string]any{
		"content": content,
		"edited":  editedAt,
	})
}

// Delete removes a comment by reference.
func (r *CommentRepository) Delete(comment *model.GalgameToolsetComment) {
	r.db.Delete(comment)
}
