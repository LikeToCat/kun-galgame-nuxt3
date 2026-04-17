package repository

import (
	"time"

	"kun-galgame-api/internal/constants"

	"gorm.io/gorm"
)

// TopicListRepository serves the paginated topic list on the home page
// (general + resource-sections variants) with sort/NSFW/category filters.
type TopicListRepository struct {
	db *gorm.DB
}

func NewTopicListRepository(db *gorm.DB) *TopicListRepository {
	return &TopicListRepository{db: db}
}

func (r *TopicListRepository) DB() *gorm.DB { return r.db }

// TopicCardRow is the joined topic + author row used by list pages.
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

// FindList returns the generic topic list page matching the given filters.
func (r *TopicListRepository) FindList(
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

// FindResourceList returns topics that belong to resource sections
// (g-seeking, g-other, t-help).
func (r *TopicListRepository) FindResourceList(
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
		Joins(`JOIN topic_section_relation tsr ON tsr.topic_id = topic.id`).
		Joins(`JOIN topic_section ts ON ts.id = tsr.topic_section_id`).
		Where("topic.status != 1").
		Where("ts.name IN ?", []string{"g-seeking", "g-other", "t-help"}).
		Group("topic.id, \"user\".name, \"user\".avatar")

	if !isNSFW {
		query = query.Where("topic.is_nsfw = false")
	}
	if category != "" && category != "all" {
		query = query.Where("topic.category = ?", category)
	}

	query.Count(&total)

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
