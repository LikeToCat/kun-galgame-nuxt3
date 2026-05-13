package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GalgameInteractionRepository owns the user→galgame interaction rows
// (likes, favorites) and the counter side-effects on galgame_local.
type GalgameInteractionRepository struct {
	db *gorm.DB
}

func NewGalgameInteractionRepository(db *gorm.DB) *GalgameInteractionRepository {
	return &GalgameInteractionRepository{db: db}
}

func (r *GalgameInteractionRepository) DB() *gorm.DB { return r.db }

// UserInteraction reports whether the user has liked / favorited a galgame.
func (r *GalgameInteractionRepository) UserInteraction(userID, galgameID int) (liked, favorited bool) {
	if userID <= 0 {
		return false, false
	}
	var lc, fc int64
	r.db.Model(&model.GalgameLike{}).
		Where("user_id = ? AND galgame_id = ?", userID, galgameID).Count(&lc)
	r.db.Model(&model.GalgameFavorite{}).
		Where("user_id = ? AND galgame_id = ?", userID, galgameID).Count(&fc)
	return lc > 0, fc > 0
}

// ToggleLike inserts/removes a like row atomically within a caller-supplied tx.
// Returns whether the result is "now liked".
//
// Lazy-creates the local stub before incrementing: pending submissions don't
// get a stub at submit time, so the first interaction (which may be the
// submitter liking their own pending row) must INSERT one or the UPDATE
// would silently affect 0 rows.
func (r *GalgameInteractionRepository) ToggleLike(tx *gorm.DB, userID, galgameID int) (liked bool) {
	var existing model.GalgameLike
	result := tx.Where("user_id = ? AND galgame_id = ?", userID, galgameID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: galgameID})
		tx.Create(&model.GalgameLike{UserID: userID, GalgameID: galgameID})
		tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("like_count", gorm.Expr("like_count + 1"))
		return true
	}

	tx.Delete(&existing)
	tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("like_count", gorm.Expr("like_count - 1"))
	return false
}

// ToggleFavorite inserts/removes a favorite row atomically within a caller tx.
// Returns whether the result is "now favorited".
//
// Same lazy-stub rationale as ToggleLike.
func (r *GalgameInteractionRepository) ToggleFavorite(tx *gorm.DB, userID, galgameID int) (favorited bool) {
	var existing model.GalgameFavorite
	result := tx.Where("user_id = ? AND galgame_id = ?", userID, galgameID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: galgameID})
		tx.Create(&model.GalgameFavorite{UserID: userID, GalgameID: galgameID})
		tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("favorite_count", gorm.Expr("favorite_count + 1"))
		return true
	}

	tx.Delete(&existing)
	tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("favorite_count", gorm.Expr("favorite_count - 1"))
	return false
}
