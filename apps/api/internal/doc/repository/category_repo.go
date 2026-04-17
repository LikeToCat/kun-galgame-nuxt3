package repository

import (
	"kun-galgame-api/internal/doc/model"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) DB() *gorm.DB { return r.db }

// FindPaginated returns categories matching an optional keyword plus total.
func (r *CategoryRepository) FindPaginated(keyword string, page, limit int) ([]model.DocCategory, int64) {
	query := r.db.Model(&model.DocCategory{})
	if keyword != "" {
		query = query.Where(
			"title ILIKE ? OR slug ILIKE ?",
			"%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var total int64
	query.Count(&total)

	var categories []model.DocCategory
	query.Order("sort_order ASC, id ASC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&categories)

	return categories, total
}

// Create inserts a new category.
func (r *CategoryRepository) Create(category *model.DocCategory) error {
	return r.db.Create(category).Error
}

// UpdateFields updates arbitrary fields on a category row.
func (r *CategoryRepository) UpdateFields(id int, updates map[string]any) {
	r.db.Model(&model.DocCategory{}).Where("id = ?", id).Updates(updates)
}

// DeleteByID deletes a category row.
func (r *CategoryRepository) DeleteByID(id int) {
	r.db.Delete(&model.DocCategory{}, id)
}
