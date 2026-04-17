package repository

import (
	"fmt"

	"kun-galgame-api/internal/galgame/model"
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type RatingRepository struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

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

// FindComments returns comments on a rating, joined with user info, oldest first.
func (r *RatingRepository) FindComments(ratingID int) []model.RatingCommentRow {
	var rows []model.RatingCommentRow
	r.db.Table("galgame_rating_comment c").
		Select(`c.id, c.content, c.user_id, c.target_user_id,
			u1.name AS user_name, u1.avatar AS user_avatar,
			u2.name AS target_name, u2.avatar AS target_avatar,
			c.created, c.updated`).
		Joins(`LEFT JOIN "user" u1 ON u1.id = c.user_id`).
		Joins(`LEFT JOIN "user" u2 ON u2.id = c.target_user_id`).
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
// Users
// ──────────────────────────────────────────

// FindUsersByIDs batch-loads user brief info.
func (r *RatingRepository) FindUsersByIDs(ids []int) map[int]userModel.UserBrief {
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

// FindUsersListByIDs returns users preserving input order semantics (by ID list).
func (r *RatingRepository) FindUsersListByIDs(ids []int) []userModel.UserBrief {
	if len(ids) == 0 {
		return []userModel.UserBrief{}
	}
	var users []userModel.UserBrief
	r.db.Where("id IN ?", ids).Find(&users)
	return users
}
