package repository

import (
	"time"

	"gorm.io/gorm"
)

type SectionRepository struct {
	db *gorm.DB
}

func NewSectionRepository(db *gorm.DB) *SectionRepository {
	return &SectionRepository{db: db}
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

type SectionTopicRow struct {
	ID           int
	Title        string
	Content      string
	View         int
	LikeCount    int
	ReplyCount   int
	Status       int
	IsNSFW       bool
	BestAnswerID *int
	UserID       int
	UserName     string
	UserAvatar   string
	Created      time.Time
}

type SectionStatRow struct {
	SectionID   int    `gorm:"column:section_id"`
	SectionName string `gorm:"column:section_name"`
	TopicCount  int64  `gorm:"column:topic_count"`
	ViewCount   int64  `gorm:"column:view_count"`
}

type LatestTopicRow struct {
	ID      int    `gorm:"column:id"`
	Title   string `gorm:"column:title"`
	Created string `gorm:"column:created"`
}

// ──────────────────────────────────────────
// Queries
// ──────────────────────────────────────────

// FindSectionTopics returns paginated topics for a section plus total count.
func (r *SectionRepository) FindSectionTopics(
	section, sortOrder string, page, limit int,
) (rows []SectionTopicRow, total int64) {
	query := r.db.Table("topic t").
		Select(`t.id, t.title, SUBSTRING(t.content, 1, 233) AS content,
			t.view, t.like_count, t.reply_count, t.status, t.is_nsfw,
			t.best_answer_id, t.user_id,
			u.name AS user_name, u.avatar AS user_avatar, t.created`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Joins("JOIN topic_section_relation tsr ON tsr.topic_id = t.id").
		Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
		Where("ts.name = ? AND t.status != 1", section)

	query.Count(&total)

	query.Order("t.created " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows)
	return
}

// FindCategoryStats returns per-section topic count + view count filtered by category.
func (r *SectionRepository) FindCategoryStats(category string) []SectionStatRow {
	var rows []SectionStatRow
	r.db.Raw(`
		SELECT ts.id AS section_id, ts.name AS section_name,
			COUNT(DISTINCT t.id) AS topic_count,
			COALESCE(SUM(t.view), 0) AS view_count
		FROM topic_section ts
		JOIN topic_section_relation tsr ON tsr.topic_section_id = ts.id
		JOIN topic t ON t.id = tsr.topic_id AND t.status != 1
			AND t.category = ?
		GROUP BY ts.id, ts.name
		ORDER BY ts.id
	`, category).Scan(&rows)
	return rows
}

// FindLatestTopicInSection returns the most recent topic in a section for the given category.
// Returns nil if no matching topic exists.
func (r *SectionRepository) FindLatestTopicInSection(sectionID int, category string) *LatestTopicRow {
	var latest LatestTopicRow
	result := r.db.Raw(`
		SELECT t.id, t.title, t.created
		FROM topic t
		JOIN topic_section_relation tsr ON tsr.topic_id = t.id
		WHERE tsr.topic_section_id = ? AND t.status != 1
			AND t.category = ?
		ORDER BY t.created DESC LIMIT 1
	`, sectionID, category).Scan(&latest)
	if result.RowsAffected == 0 {
		return nil
	}
	return &latest
}
