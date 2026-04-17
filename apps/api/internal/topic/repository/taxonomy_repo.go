package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TopicTaxonomyRepository owns the two taxonomy-style relations attached to
// a topic: tag rows + topic_tag_relation, and section rows + topic_section_relation.
type TopicTaxonomyRepository struct {
	db *gorm.DB
}

func NewTopicTaxonomyRepository(db *gorm.DB) *TopicTaxonomyRepository {
	return &TopicTaxonomyRepository{db: db}
}

func (r *TopicTaxonomyRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Tags
// ──────────────────────────────────────────

func (r *TopicTaxonomyRepository) FindOrCreateTags(names []string) ([]model.TopicTag, error) {
	var tags []model.TopicTag
	for _, name := range names {
		var tag model.TopicTag
		result := r.db.Where("name = ?", name).First(&tag)
		if result.Error == gorm.ErrRecordNotFound {
			tag = model.TopicTag{Name: name}
			if err := r.db.Create(&tag).Error; err != nil {
				return nil, err
			}
		} else if result.Error != nil {
			return nil, result.Error
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *TopicTaxonomyRepository) ReplaceTopicTags(tx *gorm.DB, topicID int, tagIDs []int) error {
	if err := tx.Where("topic_id = ?", topicID).Delete(&model.TopicTagRelation{}).Error; err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		rel := model.TopicTagRelation{TopicID: topicID, TagID: tagID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *TopicTaxonomyRepository) FindTagNamesByTopicID(topicID int) ([]string, error) {
	var names []string
	err := r.db.Table("topic_tag_relation").
		Select("topic_tag.name").
		Joins("JOIN topic_tag ON topic_tag.id = topic_tag_relation.tag_id").
		Where("topic_tag_relation.topic_id = ?", topicID).
		Pluck("topic_tag.name", &names).Error
	return names, err
}

func (r *TopicTaxonomyRepository) FindTagNamesByTopicIDs(topicIDs []int) (map[int][]string, error) {
	type row struct {
		TopicID int
		Name    string
	}
	var rows []row
	err := r.db.Table("topic_tag_relation").
		Select("topic_tag_relation.topic_id, topic_tag.name").
		Joins("JOIN topic_tag ON topic_tag.id = topic_tag_relation.tag_id").
		Where("topic_tag_relation.topic_id IN ?", topicIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]string)
	for _, r := range rows {
		result[r.TopicID] = append(result[r.TopicID], r.Name)
	}
	return result, nil
}

// CreateTopicTagRelation upserts a (topic_id, tag_id) pair (DoNothing on conflict).
func (r *TopicTaxonomyRepository) CreateTopicTagRelation(tx *gorm.DB, topicID, tagID int) error {
	rel := model.TopicTagRelation{TopicID: topicID, TagID: tagID}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error
}

// ──────────────────────────────────────────
// Sections
// ──────────────────────────────────────────

func (r *TopicTaxonomyRepository) FindSectionsByNames(names []string) ([]model.TopicSection, error) {
	var sections []model.TopicSection
	err := r.db.Where("name IN ?", names).Find(&sections).Error
	return sections, err
}

func (r *TopicTaxonomyRepository) ReplaceSectionRelations(tx *gorm.DB, topicID int, sectionIDs []int) error {
	if err := tx.Where("topic_id = ?", topicID).Delete(&model.TopicSectionRelation{}).Error; err != nil {
		return err
	}
	for _, sID := range sectionIDs {
		rel := model.TopicSectionRelation{TopicID: topicID, TopicSectionID: sID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *TopicTaxonomyRepository) FindSectionNamesByTopicIDs(topicIDs []int) (map[int][]string, error) {
	type row struct {
		TopicID int
		Name    string
	}
	var rows []row
	err := r.db.Table("topic_section_relation").
		Select("topic_section_relation.topic_id, topic_section.name").
		Joins("JOIN topic_section ON topic_section.id = topic_section_relation.topic_section_id").
		Where("topic_section_relation.topic_id IN ?", topicIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]string)
	for _, r := range rows {
		result[r.TopicID] = append(result[r.TopicID], r.Name)
	}
	return result, nil
}

func (r *TopicTaxonomyRepository) FindSectionNamesByTopicID(topicID int) ([]string, error) {
	var names []string
	err := r.db.Table("topic_section_relation").
		Select("topic_section.name").
		Joins("JOIN topic_section ON topic_section.id = topic_section_relation.topic_section_id").
		Where("topic_section_relation.topic_id = ?", topicID).
		Pluck("topic_section.name", &names).Error
	return names, err
}

// CreateSectionRelation upserts a (topic_id, section_id) pair.
func (r *TopicTaxonomyRepository) CreateSectionRelation(tx *gorm.DB, topicID, sectionID int) error {
	rel := model.TopicSectionRelation{TopicID: topicID, TopicSectionID: sectionID}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error
}

// FindSectionsByNamesTx resolves section names to rows inside the caller tx.
func (r *TopicTaxonomyRepository) FindSectionsByNamesTx(tx *gorm.DB, names []string) ([]model.TopicSection, error) {
	var sections []model.TopicSection
	err := tx.Where("name IN ?", names).Find(&sections).Error
	return sections, err
}
