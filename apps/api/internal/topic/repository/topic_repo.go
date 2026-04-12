package repository

import (
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (r *TopicRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Core CRUD
// ──────────────────────────────────────────

func (r *TopicRepository) FindByID(id int) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.First(&topic, id).Error
	return &topic, err
}

func (r *TopicRepository) Create(topic *model.Topic) error {
	return r.db.Create(topic).Error
}

func (r *TopicRepository) UpdateFields(id int, fields map[string]any) error {
	return r.db.Model(&model.Topic{}).Where("id = ?", id).Updates(fields).Error
}

func (r *TopicRepository) IncrementView(id int) error {
	return r.db.Model(&model.Topic{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1")).Error
}

// ──────────────────────────────────────────
// List query
// ──────────────────────────────────────────

type TopicCardRow struct {
	ID               int
	Title            string
	View             int
	Status           int
	IsNSFW           bool
	LikeCount        int
	ReplyCount       int
	CommentCount     int
	BestAnswerID     *int
	StatusUpdateTime time.Time
	UpvoteTime       *time.Time
	UserID           int
	UserName         string
	UserAvatar       string
}

func (r *TopicRepository) FindList(
	page, limit int,
	sortField, sortOrder, category string,
	isNSFW bool,
) ([]TopicCardRow, int64, error) {
	var rows []TopicCardRow
	var total int64

	query := r.db.Table("topic").
		Select(`topic.id, topic.title, topic.view, topic.status,
			topic.is_nsfw, topic.like_count, topic.reply_count,
			topic.comment_count, topic.best_answer_id,
			topic.status_update_time, topic.upvote_time,
			topic.user_id,
			"user".name AS user_name, "user".avatar AS user_avatar`).
		Joins(`LEFT JOIN "user" ON "user".id = topic.user_id`).
		Where("topic.status != 1")

	if !isNSFW {
		query = query.Where("topic.is_nsfw = false")
	}
	if category != "" && category != "all" {
		query = query.Where("topic.category = ?", category)
	}

	query.Count(&total)

	// Determine sort column
	orderCol := "topic.created"
	if col, ok := constants.ValidTopicSortFields[sortField]; ok {
		orderCol = "topic." + col
	} else if col, ok := constants.ValidTopicCountSortFields[sortField]; ok {
		orderCol = "topic." + col
	}
	query = query.Order(orderCol + " " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit)

	err := query.Find(&rows).Error
	return rows, total, err
}

// ──────────────────────────────────────────
// Tags
// ──────────────────────────────────────────

func (r *TopicRepository) FindOrCreateTags(names []string) ([]model.TopicTag, error) {
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

func (r *TopicRepository) ReplaceTopicTags(tx *gorm.DB, topicID int, tagIDs []int) error {
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

func (r *TopicRepository) FindTagNamesByTopicID(topicID int) ([]string, error) {
	var names []string
	err := r.db.Table("topic_tag_relation").
		Select("topic_tag.name").
		Joins("JOIN topic_tag ON topic_tag.id = topic_tag_relation.tag_id").
		Where("topic_tag_relation.topic_id = ?", topicID).
		Pluck("topic_tag.name", &names).Error
	return names, err
}

func (r *TopicRepository) FindTagNamesByTopicIDs(topicIDs []int) (map[int][]string, error) {
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

// ──────────────────────────────────────────
// Sections
// ──────────────────────────────────────────

func (r *TopicRepository) FindSectionsByNames(names []string) ([]model.TopicSection, error) {
	var sections []model.TopicSection
	err := r.db.Where("name IN ?", names).Find(&sections).Error
	return sections, err
}

func (r *TopicRepository) ReplaceSectionRelations(tx *gorm.DB, topicID int, sectionIDs []int) error {
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

func (r *TopicRepository) FindSectionNamesByTopicIDs(topicIDs []int) (map[int][]string, error) {
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

func (r *TopicRepository) FindSectionNamesByTopicID(topicID int) ([]string, error) {
	var names []string
	err := r.db.Table("topic_section_relation").
		Select("topic_section.name").
		Joins("JOIN topic_section ON topic_section.id = topic_section_relation.topic_section_id").
		Where("topic_section_relation.topic_id = ?", topicID).
		Pluck("topic_section.name", &names).Error
	return names, err
}

// ──────────────────────────────────────────
// Interaction checks
// ──────────────────────────────────────────

func (r *TopicRepository) HasUserLiked(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicLike{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserDisliked(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicDislike{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserFavorited(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicFavorite{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserUpvoted(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicUpvote{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

// ──────────────────────────────────────────
// Daily limit
// ──────────────────────────────────────────

func (r *TopicRepository) CountTodayTopicsByUser(tx *gorm.DB, uid int) (int64, error) {
	var count int64
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := tx.Model(&model.Topic{}).
		Where("user_id = ? AND created >= ?", uid, oneDayAgo).
		Count(&count).Error
	return count, err
}

// ──────────────────────────────────────────
// Poll existence check
// ──────────────────────────────────────────

func (r *TopicRepository) HasPoll(topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicPoll{}).Where("topic_id = ?", topicID).Count(&count).Error
	return count > 0, err
}
