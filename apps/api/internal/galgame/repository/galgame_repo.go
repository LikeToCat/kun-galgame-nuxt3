package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// GalgameRepository holds core local DB queries for the galgame table:
// single-row + batch local stats lookups, user brief batching, view bumps,
// and the create-stub helper used during wiki creation.
//
// Sibling repos in this package own the other concerns:
//   - GalgameInteractionRepository (interaction_repo.go)
//   - GalgameListRepository        (list_repo.go)
//   - GalgameResourceMetaRepository (resource_meta_repo.go)
//   - GalgameDetailRatingRepository (detail_rating_repo.go)
type GalgameRepository struct {
	db *gorm.DB
}

func NewGalgameRepository(db *gorm.DB) *GalgameRepository {
	return &GalgameRepository{db: db}
}

// DB returns the underlying GORM handle (for services needing transactions).
func (r *GalgameRepository) DB() *gorm.DB {
	return r.db
}

// GalgameLocalRow is a lightweight row of local stats for enriching wiki data.
type GalgameLocalRow struct {
	ID            int `gorm:"column:id"`
	LikeCount     int `gorm:"column:like_count"`
	FavoriteCount int `gorm:"column:favorite_count"`
	View          int `gorm:"column:view"`
}

// ──────────────────────────────────────────
// Stats & users (shared)
// ──────────────────────────────────────────

// FindLocal returns the local stats row for a single galgame.
func (r *GalgameRepository) FindLocal(id int) model.GalgameLocal {
	var row model.GalgameLocal
	r.db.Where("id = ?", id).First(&row)
	return row
}

// FindLocalBatch returns local stats for a list of galgame IDs.
func (r *GalgameRepository) FindLocalBatch(ids []int) map[int]GalgameLocalRow {
	if len(ids) == 0 {
		return map[int]GalgameLocalRow{}
	}
	var rows []GalgameLocalRow
	r.db.Table("galgame").Select("id, like_count, favorite_count, view").
		Where("id IN ?", ids).Scan(&rows)
	out := make(map[int]GalgameLocalRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

// IncrementView is a best-effort view bump (fired as a goroutine by caller).
func (r *GalgameRepository) IncrementView(id int) {
	r.db.Table("galgame").Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

// ──────────────────────────────────────────
// Side-effect helpers used by Create / MergePR
// ──────────────────────────────────────────

// CreateLocalStub creates the empty galgame row on the local side after wiki
// creation succeeds, inside the given transaction.
func (r *GalgameRepository) CreateLocalStub(tx *gorm.DB, galgameID int) {
	tx.Create(&model.GalgameLocal{ID: galgameID})
}
