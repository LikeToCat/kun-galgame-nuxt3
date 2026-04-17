package repository

import (
	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) DB() *gorm.DB { return r.db }

// FindByName returns a category by unique name.
func (r *CategoryRepository) FindByName(name string) (*model.GalgameWebsiteCategory, error) {
	var cat model.GalgameWebsiteCategory
	if err := r.db.Where("name = ?", name).First(&cat).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

// FindByID returns a category by ID.
func (r *CategoryRepository) FindByID(id int) (*model.GalgameWebsiteCategory, error) {
	var cat model.GalgameWebsiteCategory
	if err := r.db.First(&cat, id).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

// FindNamesByIDs returns a map[id]name for the given category IDs.
func (r *CategoryRepository) FindNamesByIDs(ids []int) map[int]string {
	if len(ids) == 0 {
		return map[int]string{}
	}
	var rows []struct {
		ID   int
		Name string
	}
	r.db.Table("galgame_website_category").
		Select("id, name").
		Where("id IN ?", ids).
		Scan(&rows)
	out := make(map[int]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out
}

// UpdateFields updates arbitrary fields on a category row.
func (r *CategoryRepository) UpdateFields(id int, updates map[string]any) {
	r.db.Model(&model.GalgameWebsiteCategory{}).Where("id = ?", id).Updates(updates)
}
