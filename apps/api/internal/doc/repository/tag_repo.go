package repository

import (
	"kun-galgame-api/internal/doc/model"

	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) DB() *gorm.DB { return r.db }

// FindPaginated returns tags matching an optional keyword plus total.
func (r *TagRepository) FindPaginated(keyword string, page, limit int) ([]model.DocTag, int64) {
	query := r.db.Model(&model.DocTag{})
	if keyword != "" {
		query = query.Where(
			"title ILIKE ? OR slug ILIKE ?",
			"%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var total int64
	query.Count(&total)

	var tags []model.DocTag
	query.Order("title ASC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&tags)

	return tags, total
}

// Create inserts a new tag.
func (r *TagRepository) Create(tag *model.DocTag) error {
	return r.db.Create(tag).Error
}

// DeleteByID deletes a tag and any related article-tag rows.
func (r *TagRepository) DeleteByID(id int) {
	r.db.Where("doc_tag_id = ?", id).Delete(&model.DocArticleTagRelation{})
	r.db.Delete(&model.DocTag{}, id)
}

// Update applies the provided fields to a tag and returns the refreshed row.
func (r *TagRepository) Update(id int, fields map[string]any) (*model.DocTag, error) {
	if err := r.db.Model(&model.DocTag{}).Where("id = ?", id).
		Updates(fields).Error; err != nil {
		return nil, err
	}
	var tag model.DocTag
	if err := r.db.First(&tag, id).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}
