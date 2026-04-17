package repository

import (
	"kun-galgame-api/internal/galgame/model"
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type ResourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

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

// FindUsersByIDs batch-loads user brief info.
func (r *ResourceRepository) FindUsersByIDs(ids []int) map[int]userModel.UserBrief {
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

// IncrementView atomically bumps the view count.
func (r *ResourceRepository) IncrementView(resourceID int) {
	r.db.Exec("UPDATE galgame_resource SET view = view + 1 WHERE id = ?", resourceID)
}

// IncrementDownload atomically bumps the download count.
func (r *ResourceRepository) IncrementDownload(resourceID int) {
	r.db.Exec("UPDATE galgame_resource SET download = download + 1 WHERE id = ?", resourceID)
}
