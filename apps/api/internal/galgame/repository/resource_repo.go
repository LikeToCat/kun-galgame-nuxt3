package repository

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// onConflictNothing is shared for batch inserts that should tolerate uniqueness
// collisions (e.g. re-inserting resource links after an edit).
var onConflictNothing = clause.OnConflict{DoNothing: true}

type ResourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

// DB exposes the underlying connection for transaction orchestration in services.
func (r *ResourceRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// CountAll returns the total number of resources.
func (r *ResourceRepository) CountAll() int64 {
	var total int64
	r.db.Table("galgame_resource").Count(&total)
	return total
}

// ListPaginated returns resources ordered by created DESC.
func (r *ResourceRepository) ListPaginated(page, limit int) []model.GalgameResourceRow {
	offset := (page - 1) * limit
	var rows []model.GalgameResourceRow
	r.db.Table("galgame_resource").
		Order("created DESC").
		Offset(offset).Limit(limit).
		Scan(&rows)
	return rows
}

// FindByID returns a single resource row. Returns false if not found.
func (r *ResourceRepository) FindByID(id int) (model.GalgameResourceRow, bool) {
	var row model.GalgameResourceRow
	if err := r.db.Table("galgame_resource").Where("id = ?", id).Scan(&row).Error; err != nil || row.ID == 0 {
		return row, false
	}
	return row, true
}

// FindByGalgameID returns all resources for a galgame, ordered by created DESC.
func (r *ResourceRepository) FindByGalgameID(galgameID int) []model.GalgameResourceRow {
	var rows []model.GalgameResourceRow
	r.db.Table("galgame_resource").
		Where("galgame_id = ?", galgameID).
		Order("created DESC").
		Scan(&rows)
	return rows
}

// FindRecommendations returns other resources in the same galgame, sorted by
// like_count DESC, limited to `limit`.
func (r *ResourceRepository) FindRecommendations(galgameID, excludeID, limit int) []model.GalgameResourceRow {
	var rows []model.GalgameResourceRow
	r.db.Table("galgame_resource").
		Where("galgame_id = ? AND id != ?", galgameID, excludeID).
		Order("like_count DESC").
		Limit(limit).
		Scan(&rows)
	return rows
}

// FindLinks returns all download URLs for a resource.
func (r *ResourceRepository) FindLinks(resourceID int) []string {
	type linkRow struct {
		URL string `gorm:"column:url"`
	}
	var links []linkRow
	r.db.Table("galgame_resource_link").
		Where("galgame_resource_id = ?", resourceID).
		Scan(&links)
	out := make([]string, len(links))
	for i, l := range links {
		out[i] = l.URL
	}
	return out
}

// AggregateByGalgame returns DISTINCT (platform, language, type) tuples.
func (r *ResourceRepository) AggregateByGalgame(galgameID int) []model.ResourceAggregate {
	var aggs []model.ResourceAggregate
	r.db.Table("galgame_resource").
		Select("DISTINCT platform, language, type").
		Where("galgame_id = ?", galgameID).
		Scan(&aggs)
	return aggs
}

// IsLikedBy checks whether a user has liked a given resource.
func (r *ResourceRepository) IsLikedBy(resourceID, userID int) bool {
	if userID <= 0 {
		return false
	}
	var cnt int64
	r.db.Table("galgame_resource_like").
		Where("galgame_resource_id = ? AND user_id = ?", resourceID, userID).
		Count(&cnt)
	return cnt > 0
}

// FindGalgameView returns the local galgame.view counter.
func (r *ResourceRepository) FindGalgameView(galgameID int) int {
	var view int
	r.db.Table("galgame").Select("view").Where("id = ?", galgameID).Scan(&view)
	return view
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// IncrementView atomically bumps the view count.
func (r *ResourceRepository) IncrementView(resourceID int) {
	r.db.Exec("UPDATE galgame_resource SET view = view + 1 WHERE id = ?", resourceID)
}

// IncrementDownload atomically bumps the download count.
func (r *ResourceRepository) IncrementDownload(resourceID int) {
	r.db.Exec("UPDATE galgame_resource SET download = download + 1 WHERE id = ?", resourceID)
}

// Create inserts a new galgame_resource row within the given tx.
func (r *ResourceRepository) Create(tx *gorm.DB, res *model.GalgameResource) error {
	return tx.Create(res).Error
}

// ReplaceProviders overwrites the text[] provider column for a resource.
// GORM has no native array type, so this uses raw SQL.
func (r *ResourceRepository) ReplaceProviders(tx *gorm.DB, resourceID int, providers []string) error {
	return tx.Exec(
		"UPDATE galgame_resource SET provider = ? WHERE id = ?",
		providers, resourceID,
	).Error
}

// ReplaceProviderNames overwrites the jsonb provider_name column with the
// given display labels (e.g. ["百度网盘", "OneDrive"]). Stored as JSON text;
// the read side either passes through the raw bytes or unmarshals to []string.
func (r *ResourceRepository) ReplaceProviderNames(tx *gorm.DB, resourceID int, names []string) error {
	if names == nil {
		names = []string{}
	}
	encoded, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Exec(
		"UPDATE galgame_resource SET provider_name = ?::jsonb WHERE id = ?",
		string(encoded), resourceID,
	).Error
}

// CreateLinks inserts resource_link rows, skipping (resource_id, url) duplicates.
func (r *ResourceRepository) CreateLinks(tx *gorm.DB, resourceID int, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	links := make([]model.GalgameResourceLink, len(urls))
	for i, u := range urls {
		links[i] = model.GalgameResourceLink{GalgameResourceID: resourceID, URL: u}
	}
	return tx.Clauses(onConflictNothing).Create(&links).Error
}

// DeleteLinks removes all links for a resource.
func (r *ResourceRepository) DeleteLinks(tx *gorm.DB, resourceID int) error {
	return tx.Where("galgame_resource_id = ?", resourceID).
		Delete(&model.GalgameResourceLink{}).Error
}

// UpdateFields patches arbitrary columns on a resource row (used for edit endpoint).
// Keys are column names (not struct field names).
func (r *ResourceRepository) UpdateFields(tx *gorm.DB, resourceID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Table("galgame_resource").Where("id = ?", resourceID).
		Updates(fields).Error
}

// UpdateStatus sets the status column (0 = valid, 1 = expired).
func (r *ResourceRepository) UpdateStatus(tx *gorm.DB, resourceID, status int) error {
	return tx.Table("galgame_resource").Where("id = ?", resourceID).
		Update("status", status).Error
}

// DeleteByID removes a resource by primary key. Cascades to links/likes via FK.
func (r *ResourceRepository) DeleteByID(tx *gorm.DB, resourceID int) error {
	return tx.Where("id = ?", resourceID).Delete(&model.GalgameResource{}).Error
}

// FindLike returns the like row for (resource, user), or ok=false.
func (r *ResourceRepository) FindLike(tx *gorm.DB, resourceID, userID int) (model.GalgameResourceLike, bool) {
	var like model.GalgameResourceLike
	err := tx.Where("galgame_resource_id = ? AND user_id = ?", resourceID, userID).
		First(&like).Error
	if err != nil {
		return like, false
	}
	return like, true
}

// CreateLike adds a like row for (resource, user).
func (r *ResourceRepository) CreateLike(tx *gorm.DB, resourceID, userID int) error {
	return tx.Create(&model.GalgameResourceLike{
		GalgameResourceID: resourceID, UserID: userID,
	}).Error
}

// DeleteLike removes a specific like row.
func (r *ResourceRepository) DeleteLike(tx *gorm.DB, like model.GalgameResourceLike) error {
	return tx.Delete(&like).Error
}

// AdjustLikeCount adjusts the resource.like_count counter by delta (+1/-1).
func (r *ResourceRepository) AdjustLikeCount(tx *gorm.DB, resourceID, delta int) error {
	return tx.Table("galgame_resource").Where("id = ?", resourceID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// AdjustLocalResourceCount adjusts galgame.resource_count by delta.
func (r *ResourceRepository) AdjustLocalResourceCount(tx *gorm.DB, galgameID, delta int) error {
	return tx.Table("galgame").Where("id = ?", galgameID).
		Update("resource_count", gorm.Expr("resource_count + ?", delta)).Error
}
