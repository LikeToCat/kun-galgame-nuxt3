package repository

import (
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type ResourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r *ResourceRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// FindByID returns a single resource.
func (r *ResourceRepository) FindByID(id int) (*model.GalgameToolsetResource, error) {
	var resource model.GalgameToolsetResource
	if err := r.db.First(&resource, id).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

// FindByToolset returns all resources for a toolset ordered by created DESC.
func (r *ResourceRepository) FindByToolset(toolsetID int) []model.GalgameToolsetResource {
	var resources []model.GalgameToolsetResource
	r.db.Where("toolset_id = ?", toolsetID).Order("created DESC").Find(&resources)
	return resources
}

// FindS3ByToolsetTx returns all s3-type resources for a toolset (within a tx).
func (r *ResourceRepository) FindS3ByToolsetTx(tx *gorm.DB, toolsetID int) []model.GalgameToolsetResource {
	var resources []model.GalgameToolsetResource
	tx.Where("toolset_id = ? AND type = 's3'", toolsetID).Find(&resources)
	return resources
}

// DownloadSum returns the sum of download counts for a toolset.
func (r *ResourceRepository) DownloadSum(toolsetID int) int64 {
	var sum int64
	r.db.Model(&model.GalgameToolsetResource{}).
		Where("toolset_id = ?", toolsetID).
		Select("COALESCE(SUM(download), 0)").Scan(&sum)
	return sum
}

// DownloadSumsForToolsets returns map[toolsetID]downloadSum for a batch.
func (r *ResourceRepository) DownloadSumsForToolsets(toolsetIDs []int) map[int]int {
	if len(toolsetIDs) == 0 {
		return map[int]int{}
	}
	type row struct {
		ToolsetID int
		Total     int
	}
	var rows []row
	r.db.Model(&model.GalgameToolsetResource{}).
		Select("toolset_id, COALESCE(SUM(download), 0) AS total").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.ToolsetID] = r.Total
	}
	return out
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new resource (within a tx).
func (r *ResourceRepository) Create(tx *gorm.DB, resource *model.GalgameToolsetResource) error {
	return tx.Create(resource).Error
}

// UpdateFields updates arbitrary fields on a resource.
func (r *ResourceRepository) UpdateFields(resource *model.GalgameToolsetResource, updates map[string]any) {
	r.db.Model(resource).Updates(updates)
}

// IncrementDownload bumps the download count by 1 (async, no tx).
func (r *ResourceRepository) IncrementDownload(id int) {
	r.db.Model(&model.GalgameToolsetResource{}).Where("id = ?", id).
		Update("download", gorm.Expr("download + 1"))
}

// Delete removes a resource.
func (r *ResourceRepository) Delete(resource *model.GalgameToolsetResource) {
	r.db.Delete(resource)
}

// SetEditedNow sets the edited timestamp to now() on a resource.
func (r *ResourceRepository) SetEditedNow(resource *model.GalgameToolsetResource, now time.Time) {
	r.db.Model(resource).Update("edited", now)
}

// FindUser returns a single user brief row used for the resource detail view.
func (r *ResourceRepository) FindUser(userID int) userModel.UserBrief {
	var u userModel.UserBrief
	r.db.Where("id = ?", userID).First(&u)
	return u
}
