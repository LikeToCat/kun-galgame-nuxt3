package repository

import (
	"fmt"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type RatingRepository struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

// DB exposes the connection for service-owned transactions.
func (r *RatingRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Reads — single rating
// ──────────────────────────────────────────

// FindByID returns a single rating row. Returns false if not found.
func (r *RatingRepository) FindByID(id int) (model.GalgameRatingRow, bool) {
	var row model.GalgameRatingRow
	if err := r.db.Table("galgame_rating").Where("id = ?", id).Scan(&row).Error; err != nil || row.ID == 0 {
		return row, false
	}
	return row, true
}

// FindLikerIDs returns the user IDs who liked a given rating.
func (r *RatingRepository) FindLikerIDs(ratingID int) []int {
	type row struct {
		UserID int `gorm:"column:user_id"`
	}
	var rows []row
	r.db.Table("galgame_rating_like").Select("user_id").
		Where("galgame_rating_id = ?", ratingID).Scan(&rows)
	out := make([]int, len(rows))
	for i, r := range rows {
		out[i] = r.UserID
	}
	return out
}

// FindComments returns comments on a rating, oldest first. Identity is
// hydrated at the service layer via userclient.
func (r *RatingRepository) FindComments(ratingID int) []model.RatingCommentRow {
	var rows []model.RatingCommentRow
	r.db.Table("galgame_rating_comment c").
		Select(`c.id, c.content, c.user_id, c.target_user_id,
			c.created, c.updated`).
		Where("c.galgame_rating_id = ?", ratingID).
		Order("c.created ASC").
		Scan(&rows)
	return rows
}

// GalgameRatingStats returns SUM(overall) and COUNT(*) for a galgame.
func (r *RatingRepository) GalgameRatingStats(galgameID int) (sum, count int64) {
	r.db.Table("galgame_rating").Select("COALESCE(SUM(overall), 0)").
		Where("galgame_id = ?", galgameID).Scan(&sum)
	r.db.Table("galgame_rating").
		Where("galgame_id = ?", galgameID).Count(&count)
	return
}

// IncrementView atomically bumps the view counter (best-effort).
func (r *RatingRepository) IncrementView(ratingID int) {
	go r.db.Table("galgame_rating").Where("id = ?", ratingID).
		Update("view", gorm.Expr("view + 1"))
}

// ──────────────────────────────────────────
// Reads — list with filters
// ──────────────────────────────────────────

// ListPaginated applies the filter and returns (rows, total).
func (r *RatingRepository) ListPaginated(f model.RatingFilter) ([]model.GalgameRatingRow, int64) {
	query := r.db.Table("galgame_rating r")
	if f.SpoilerLevel != "" && f.SpoilerLevel != "all" {
		query = query.Where("r.spoiler_level = ?", f.SpoilerLevel)
	}
	if f.PlayStatus != "" && f.PlayStatus != "all" {
		query = query.Where("r.play_status = ?", f.PlayStatus)
	}
	if f.GalgameType != "" && f.GalgameType != "all" {
		query = query.Where("r.galgame_type @> ?", fmt.Sprintf(`["%s"]`, f.GalgameType))
	}

	var total int64
	query.Count(&total)

	orderCol := "r.created"
	switch f.SortField {
	case "view":
		orderCol = "r.view"
	case "overall":
		orderCol = "r.overall"
	}

	var rows []model.GalgameRatingRow
	query.Select("r.*").
		Order(orderCol + " " + f.SortOrder).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)
	return rows, total
}

// ──────────────────────────────────────────
// Writes — rating
// ──────────────────────────────────────────

// ExistsByUserGalgame reports whether the user has already rated this galgame.
func (r *RatingRepository) ExistsByUserGalgame(galgameID, userID int) bool {
	var cnt int64
	r.db.Table("galgame_rating").
		Where("galgame_id = ? AND user_id = ?", galgameID, userID).
		Count(&cnt)
	return cnt > 0
}

// FindRatingForWrite returns the writable rating model — used for permission
// and length checks before update/delete.
func (r *RatingRepository) FindRatingForWrite(id int) (*model.GalgameRating, error) {
	var rating model.GalgameRating
	err := r.db.First(&rating, id).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

// Create inserts a new galgame_rating row.
func (r *RatingRepository) Create(tx *gorm.DB, rating *model.GalgameRating) error {
	return tx.Create(rating).Error
}

// Update patches arbitrary columns on a rating row.
func (r *RatingRepository) Update(tx *gorm.DB, ratingID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Table("galgame_rating").Where("id = ?", ratingID).Updates(fields).Error
}

// DeleteByID removes a rating (cascade clears likes & comments).
func (r *RatingRepository) DeleteByID(tx *gorm.DB, ratingID int) error {
	return tx.Where("id = ?", ratingID).Delete(&model.GalgameRating{}).Error
}

// FindGalgameOwner reads galgame.user_id used for delete permission checks.
func (r *RatingRepository) FindGalgameOwner(galgameID int) int {
	var uid int
	r.db.Table("galgame").Select("user_id").Where("id = ?", galgameID).Scan(&uid)
	return uid
}

// ──────────────────────────────────────────
// Writes — rating like
// ──────────────────────────────────────────

// FindLike returns the like row for (rating, user), or ok=false.
func (r *RatingRepository) FindLike(tx *gorm.DB, ratingID, userID int) (model.GalgameRatingLike, bool) {
	var like model.GalgameRatingLike
	err := tx.Where("galgame_rating_id = ? AND user_id = ?", ratingID, userID).
		First(&like).Error
	if err != nil {
		return like, false
	}
	return like, true
}

// CreateLike inserts a like row.
func (r *RatingRepository) CreateLike(tx *gorm.DB, ratingID, userID int) error {
	return tx.Create(&model.GalgameRatingLike{
		GalgameRatingID: ratingID, UserID: userID,
	}).Error
}

// DeleteLike removes a specific like row.
func (r *RatingRepository) DeleteLike(tx *gorm.DB, like model.GalgameRatingLike) error {
	return tx.Delete(&like).Error
}

// AdjustLikeCount adjusts galgame_rating.like_count by delta.
func (r *RatingRepository) AdjustLikeCount(tx *gorm.DB, ratingID, delta int) error {
	return tx.Table("galgame_rating").Where("id = ?", ratingID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// ──────────────────────────────────────────
// Writes — rating comment
// ──────────────────────────────────────────

// CreateComment inserts a new comment row.
func (r *RatingRepository) CreateComment(tx *gorm.DB, c *model.GalgameRatingComment) error {
	return tx.Create(c).Error
}

// FindCommentByID returns a comment for permission checks.
func (r *RatingRepository) FindCommentByID(id int) (*model.GalgameRatingComment, error) {
	var c model.GalgameRatingComment
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCommentContent patches the content field.
func (r *RatingRepository) UpdateCommentContent(tx *gorm.DB, commentID int, content string) error {
	return tx.Table("galgame_rating_comment").Where("id = ?", commentID).
		Update("content", content).Error
}

// DeleteCommentByID removes a comment.
func (r *RatingRepository) DeleteCommentByID(tx *gorm.DB, commentID int) error {
	return tx.Where("id = ?", commentID).Delete(&model.GalgameRatingComment{}).Error
}

// FindRatingGalgameID returns the galgame_id for a rating (used by comment notifications).
func (r *RatingRepository) FindRatingGalgameID(ratingID int) int {
	var gid int
	r.db.Table("galgame_rating").Select("galgame_id").Where("id = ?", ratingID).Scan(&gid)
	return gid
}
