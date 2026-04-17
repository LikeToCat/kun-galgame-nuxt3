package repository

import (
	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Tag reads
// ──────────────────────────────────────────

// FindAll returns all tags ordered by id ASC.
func (r *TagRepository) FindAll() []model.GalgameWebsiteTag {
	var tags []model.GalgameWebsiteTag
	r.db.Order("id ASC").Find(&tags)
	return tags
}

// FindByName returns a tag by its unique name.
func (r *TagRepository) FindByName(name string) (*model.GalgameWebsiteTag, error) {
	var tag model.GalgameWebsiteTag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// ──────────────────────────────────────────
// Tag writes
// ──────────────────────────────────────────

// Create inserts a new tag row.
func (r *TagRepository) Create(tag *model.GalgameWebsiteTag) error {
	return r.db.Create(tag).Error
}

// UpdateFields updates arbitrary fields on a tag row.
func (r *TagRepository) UpdateFields(id int, updates map[string]any) {
	r.db.Model(&model.GalgameWebsiteTag{}).Where("id = ?", id).Updates(updates)
}

// DeleteByID deletes a tag row and all its relation rows.
func (r *TagRepository) DeleteByID(id int) {
	r.db.Where("tag_id = ?", id).Delete(&model.GalgameWebsiteTagRelation{})
	r.db.Delete(&model.GalgameWebsiteTag{}, id)
}

// ──────────────────────────────────────────
// Relation reads
// ──────────────────────────────────────────

// FindRelationsByTagID returns all relation rows for a tag.
func (r *TagRepository) FindRelationsByTagID(tagID int) []model.GalgameWebsiteTagRelation {
	var rels []model.GalgameWebsiteTagRelation
	r.db.Where("galgame_website_tag_id = ?", tagID).Find(&rels)
	return rels
}

// FindRelationsByWebsiteWithTag returns relations for a website preloading the Tag.
func (r *TagRepository) FindRelationsByWebsiteWithTag(websiteID int) []model.GalgameWebsiteTagRelation {
	var rels []model.GalgameWebsiteTagRelation
	r.db.Where("galgame_website_id = ?", websiteID).
		Preload("Tag").Find(&rels)
	return rels
}

// LevelSumsByWebsiteIDs returns a map[websiteID] -> sum of tag levels for the given websites.
func (r *TagRepository) LevelSumsByWebsiteIDs(websiteIDs []int) map[int]int {
	if len(websiteIDs) == 0 {
		return map[int]int{}
	}
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var rows []tagSum
	r.db.Table("galgame_website_tag_relation r").
		Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
		Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
		Where("r.galgame_website_id IN ?", websiteIDs).
		Group("r.galgame_website_id").Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.WebsiteID] = r.Total
	}
	return out
}

// LevelSumsAll returns the map of tag level sums for every website.
func (r *TagRepository) LevelSumsAll() map[int]int {
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var rows []tagSum
	r.db.Table("galgame_website_tag_relation r").
		Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
		Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
		Group("r.galgame_website_id").Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.WebsiteID] = r.Total
	}
	return out
}

// ──────────────────────────────────────────
// Relation writes
// ──────────────────────────────────────────

// ReplaceWebsiteTagRelations deletes existing website↔tag rows and re-adds the given tag IDs.
// Call inside a tx.
func (r *TagRepository) ReplaceWebsiteTagRelations(tx *gorm.DB, websiteID int, tagIDs []int) {
	tx.Where("website_id = ?", websiteID).Delete(&model.GalgameWebsiteTagRelation{})
	for _, tagID := range tagIDs {
		tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
			GalgameWebsiteID: websiteID, GalgameWebsiteTagID: tagID,
		})
	}
}

// InsertWebsiteTagRelations inserts website↔tag rows for newly-created website (inside a tx).
func (r *TagRepository) InsertWebsiteTagRelations(tx *gorm.DB, websiteID int, tagIDs []int) {
	for _, tagID := range tagIDs {
		tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
			GalgameWebsiteID: websiteID, GalgameWebsiteTagID: tagID,
		})
	}
}
