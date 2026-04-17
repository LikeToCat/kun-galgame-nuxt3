package repository

import (
	"time"

	"gorm.io/gorm"
)

type HomeRepository struct {
	db *gorm.DB
}

func NewHomeRepository(db *gorm.DB) *HomeRepository {
	return &HomeRepository{db: db}
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

type GalgameLocalRow struct {
	ID        int `gorm:"column:id"`
	View      int `gorm:"column:view"`
	LikeCount int `gorm:"column:like_count"`
}

type UserInfoRow struct {
	ID     int    `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Avatar string `gorm:"column:avatar"`
}

type ResourcePLRow struct {
	GalgameID int    `gorm:"column:galgame_id"`
	Platform  string `gorm:"column:platform"`
	Language  string `gorm:"column:language"`
}

type TopicRow struct {
	ID               int        `gorm:"column:id"`
	Title            string     `gorm:"column:title"`
	View             int        `gorm:"column:view"`
	IsNSFW           bool       `gorm:"column:is_nsfw"`
	Status           int        `gorm:"column:status"`
	LikeCount        int        `gorm:"column:like_count"`
	ReplyCount       int        `gorm:"column:reply_count"`
	CommentCount     int        `gorm:"column:comment_count"`
	BestAnswerID     *int       `gorm:"column:best_answer_id"`
	UpvoteTime       *time.Time `gorm:"column:upvote_time"`
	StatusUpdateTime time.Time  `gorm:"column:status_update_time"`
	UserID           int        `gorm:"column:user_id"`
	UserName         string     `gorm:"column:user_name"`
	UserAvatar       string     `gorm:"column:user_avatar"`
}

type SectionRelationRow struct {
	TopicID     int    `gorm:"column:topic_id"`
	SectionName string `gorm:"column:name"`
}

type TagRelationRow struct {
	TopicID int    `gorm:"column:topic_id"`
	TagName string `gorm:"column:name"`
}

// ──────────────────────────────────────────
// Queries
// ──────────────────────────────────────────

// FindRecentGalgames returns the N most recently updated local galgames.
func (r *HomeRepository) FindRecentGalgames(limit int) ([]GalgameLocalRow, error) {
	var rows []GalgameLocalRow
	err := r.db.Table("galgame").
		Select("id, view, like_count").
		Order("updated DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// FindUsersByIDs fetches (id, name, avatar) for the given user IDs.
func (r *HomeRepository) FindUsersByIDs(ids []int) []UserInfoRow {
	if len(ids) == 0 {
		return nil
	}
	var users []UserInfoRow
	r.db.Table(`"user"`).
		Select("id, name, avatar").
		Where("id IN ?", ids).Scan(&users)
	return users
}

// FindResourcePlatformLanguage returns (galgame_id, platform, language)
// rows for the given galgame IDs.
func (r *HomeRepository) FindResourcePlatformLanguage(galgameIDs []int) []ResourcePLRow {
	if len(galgameIDs) == 0 {
		return nil
	}
	var resources []ResourcePLRow
	r.db.Table("galgame_resource").
		Select("galgame_id, platform, language").
		Where("galgame_id IN ?", galgameIDs).
		Find(&resources)
	return resources
}

// FindHomeTopics returns the most recent topics (excluding some sections)
// updated within the last 3 months.
func (r *HomeRepository) FindHomeTopics(limit int, isSFW bool) ([]TopicRow, error) {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	excludedSections := []string{"g-seeking", "g-other", "t-help"}

	query := r.db.Table("topic").
		Select(`topic.id, topic.title, topic.view, topic.is_nsfw, topic.status,
			topic.like_count, topic.reply_count, topic.comment_count,
			topic.best_answer_id, topic.upvote_time, topic.status_update_time,
			topic.user_id, "user".name AS user_name, "user".avatar AS user_avatar`).
		Joins(`JOIN "user" ON "user".id = topic.user_id`).
		Where("topic.status != 1").
		Where(`topic.id NOT IN (
			SELECT tsr.topic_id FROM topic_section_relation tsr
			JOIN topic_section ts ON ts.id = tsr.topic_section_id
			WHERE ts.name IN ?
		)`, excludedSections).
		Where(`(topic.edited >= ? OR (topic.edited IS NULL AND topic.created >= ?))`, threeMonthsAgo, threeMonthsAgo).
		Order("topic.status_update_time DESC").
		Limit(limit)

	if isSFW {
		query = query.Where("topic.is_nsfw = false")
	}

	var rows []TopicRow
	err := query.Find(&rows).Error
	return rows, err
}

// FindTopicSections returns section names grouped by topic ID.
func (r *HomeRepository) FindTopicSections(topicIDs []int) []SectionRelationRow {
	if len(topicIDs) == 0 {
		return nil
	}
	var sections []SectionRelationRow
	r.db.Table("topic_section_relation tsr").
		Select("tsr.topic_id, ts.name").
		Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
		Where("tsr.topic_id IN ?", topicIDs).
		Find(&sections)
	return sections
}

// FindTopicTags returns tag names grouped by topic ID.
func (r *HomeRepository) FindTopicTags(topicIDs []int) []TagRelationRow {
	if len(topicIDs) == 0 {
		return nil
	}
	var tags []TagRelationRow
	r.db.Table("topic_tag_relation ttr").
		Select("ttr.topic_id, tt.name").
		Joins("JOIN topic_tag tt ON tt.id = ttr.tag_id").
		Where("ttr.topic_id IN ?", topicIDs).
		Find(&tags)
	return tags
}
