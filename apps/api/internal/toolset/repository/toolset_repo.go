package repository

import (
	"time"

	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type ToolsetRepository struct {
	db *gorm.DB
}

func NewToolsetRepository(db *gorm.DB) *ToolsetRepository {
	return &ToolsetRepository{db: db}
}

func (r *ToolsetRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Filters for the list query
// ──────────────────────────────────────────

// ListFilters holds the optional filter parameters used by List/Count.
type ListFilters struct {
	Type     string
	Language string
	Platform string
	Version  string
}

// ListOptions holds sort + pagination parameters.
type ListOptions struct {
	SortField string // whitelisted column name
	SortOrder string // "ASC" or "DESC"
	Offset    int
	Limit     int
}

// buildListQuery returns a pre-filtered *gorm.DB with `status != 1` applied.
func (r *ToolsetRepository) buildListQuery(f ListFilters) *gorm.DB {
	q := r.db.Model(&model.GalgameToolset{}).Where("status != 1")
	if f.Type != "" && f.Type != "all" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Language != "" && f.Language != "all" {
		q = q.Where("language = ?", f.Language)
	}
	if f.Platform != "" && f.Platform != "all" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.Version != "" && f.Version != "all" {
		q = q.Where("version = ?", f.Version)
	}
	return q
}

// CountFiltered counts toolsets matching the filters.
func (r *ToolsetRepository) CountFiltered(f ListFilters) int64 {
	var total int64
	r.buildListQuery(f).Count(&total)
	return total
}

// ListFiltered returns toolsets matching the filters, sorted + paginated.
func (r *ToolsetRepository) ListFiltered(f ListFilters, o ListOptions) []model.GalgameToolset {
	var toolsets []model.GalgameToolset
	r.buildListQuery(f).
		Order(o.SortField + " " + o.SortOrder).
		Offset(o.Offset).Limit(o.Limit).
		Find(&toolsets)
	return toolsets
}

// ──────────────────────────────────────────
// Single-row lookups
// ──────────────────────────────────────────

// FindByID returns a toolset by ID.
func (r *ToolsetRepository) FindByID(id int) (*model.GalgameToolset, error) {
	var toolset model.GalgameToolset
	if err := r.db.First(&toolset, id).Error; err != nil {
		return nil, err
	}
	return &toolset, nil
}

// FindByIDTx is the transactional variant of FindByID.
func (r *ToolsetRepository) FindByIDTx(tx *gorm.DB, id int) (*model.GalgameToolset, error) {
	var toolset model.GalgameToolset
	if err := tx.First(&toolset, id).Error; err != nil {
		return nil, err
	}
	return &toolset, nil
}

// ──────────────────────────────────────────
// Writes (transactional)
// ──────────────────────────────────────────

// Create inserts a new toolset. Call inside a tx.
func (r *ToolsetRepository) Create(tx *gorm.DB, toolset *model.GalgameToolset) error {
	return tx.Create(toolset).Error
}

// UpdateFields updates arbitrary fields on a toolset row.
func (r *ToolsetRepository) UpdateFields(tx *gorm.DB, id int, updates map[string]any) {
	tx.Model(&model.GalgameToolset{}).Where("id = ?", id).Updates(updates)
}

// IncrementView bumps view by 1 (used by GetDetail, no tx).
func (r *ToolsetRepository) IncrementView(id int) {
	r.db.Model(&model.GalgameToolset{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

// UpdateResourceTime refreshes resource_update_time (inside a tx).
func (r *ToolsetRepository) UpdateResourceTime(tx *gorm.DB, id int, now time.Time) {
	tx.Model(&model.GalgameToolset{}).Where("id = ?", id).
		Update("resource_update_time", now)
}

// DeleteByID deletes a toolset (inside a tx).
func (r *ToolsetRepository) DeleteByID(tx *gorm.DB, id int) {
	tx.Delete(&model.GalgameToolset{}, id)
}

// ──────────────────────────────────────────
// Alias
// ──────────────────────────────────────────

// FindAliases returns all aliases for a toolset.
func (r *ToolsetRepository) FindAliases(toolsetID int) []model.GalgameToolsetAlias {
	var aliases []model.GalgameToolsetAlias
	r.db.Where("toolset_id = ?", toolsetID).Find(&aliases)
	return aliases
}

// ReplaceAliases deletes existing aliases and inserts the given names (within a tx).
// Empty/whitespace-only names are skipped.
func (r *ToolsetRepository) ReplaceAliases(tx *gorm.DB, toolsetID int, aliases []string) {
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetAlias{})
	for _, name := range aliases {
		if name == "" {
			continue
		}
		tx.Create(&model.GalgameToolsetAlias{
			Name:      name,
			ToolsetID: toolsetID,
		})
	}
}

// ──────────────────────────────────────────
// Contributor
// ──────────────────────────────────────────

// FindContributorIDs returns the user_id list of contributors for a toolset.
// Identity (name / avatar) is OAuth-owned — callers hydrate via pkg/userclient.
func (r *ToolsetRepository) FindContributorIDs(toolsetID int) []int {
	var ids []int
	r.db.Model(&model.GalgameToolsetContributor{}).
		Where("toolset_id = ?", toolsetID).
		Pluck("user_id", &ids)
	return ids
}

// AddContributor adds the given user as a contributor (inside a tx),
// ignoring if the pair already exists.
func (r *ToolsetRepository) AddContributor(tx *gorm.DB, toolsetID, userID int) {
	var cnt int64
	tx.Model(&model.GalgameToolsetContributor{}).
		Where("toolset_id = ? AND user_id = ?", toolsetID, userID).
		Count(&cnt)
	if cnt > 0 {
		return
	}
	tx.Create(&model.GalgameToolsetContributor{
		ToolsetID: toolsetID,
		UserID:    userID,
	})
}

// ──────────────────────────────────────────
// Cleanup used by Delete
// ──────────────────────────────────────────

// DeleteAllRelated deletes all child rows for a toolset. Call inside a tx.
func (r *ToolsetRepository) DeleteAllRelated(tx *gorm.DB, toolsetID int) {
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetAlias{})
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetContributor{})
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetPracticality{})
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetResource{})
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetComment{})
	tx.Where("toolset_id = ?", toolsetID).Delete(&model.GalgameToolsetCategoryRelation{})
}
