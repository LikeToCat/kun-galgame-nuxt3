package repository

import (
	"kun-galgame-api/internal/rss/dto"

	"gorm.io/gorm"
)

type RSSRepository struct {
	db *gorm.DB
}

func NewRSSRepository(db *gorm.DB) *RSSRepository {
	return &RSSRepository{db: db}
}

// FindRecentSFWTopics returns the 10 most recent SFW topics for the RSS feed.
func (r *RSSRepository) FindRecentSFWTopics() []dto.TopicRSSItem {
	var topics []dto.TopicRSSItem
	r.db.Table("topic t").
		Select(`t.id, t.title, SUBSTRING(t.content, 1, 233) AS description,
			t.user_id, u.name AS user_name, t.created`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1 AND t.is_nsfw = false").
		Order("t.created DESC").
		Limit(10).
		Find(&topics)
	return topics
}

// RecentGalgameRow is the local-only projection used to seed the galgame RSS:
// IDs and creation timestamp; metadata is fetched from wiki separately.
type RecentGalgameRow struct {
	ID      int    `gorm:"column:id"`
	Created string `gorm:"column:created"`
}

// FindRecentGalgameIDs returns the most recent local galgame stub IDs.
// Metadata (name, banner, user info) is resolved via the wiki client.
func (r *RSSRepository) FindRecentGalgameIDs(limit int) []RecentGalgameRow {
	var rows []RecentGalgameRow
	r.db.Table("galgame").
		Select("id, created").
		Order("created DESC").
		Limit(limit).
		Scan(&rows)
	return rows
}
