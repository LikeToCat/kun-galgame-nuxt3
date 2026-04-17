package repository

import (
	"gorm.io/gorm"
)

type RankingRepository struct {
	db *gorm.DB
}

func NewRankingRepository(db *gorm.DB) *RankingRepository {
	return &RankingRepository{db: db}
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

type GalgameLocalRow struct {
	ID    int `gorm:"column:id"`
	Value int `gorm:"column:value"`
}

type TopicRankingRow struct {
	ID         int    `gorm:"column:id"`
	Title      string `gorm:"column:title"`
	UserID     int    `gorm:"column:user_id"`
	UserName   string `gorm:"column:user_name"`
	UserAvatar string `gorm:"column:user_avatar"`
	Value      int    `gorm:"column:value"`
}

type UserRankingRow struct {
	ID     int    `gorm:"column:id" json:"id"`
	Name   string `gorm:"column:name" json:"name"`
	Avatar string `gorm:"column:avatar" json:"avatar"`
	Bio    string `gorm:"column:bio" json:"bio"`
	Value  int    `gorm:"column:value" json:"value"`
}

type UserInfoRow struct {
	ID     int    `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Avatar string `gorm:"column:avatar"`
}

// ──────────────────────────────────────────
// Queries
// ──────────────────────────────────────────

// FindGalgameLocal returns (id, sort_value) pairs from the galgame table
// sorted by the requested field.
func (r *RankingRepository) FindGalgameLocal(sortField, sortOrder string, page, limit int) []GalgameLocalRow {
	var rows []GalgameLocalRow
	r.db.Table("galgame").
		Select("id, "+sortField+" AS value").
		Order(sortField + " " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows)
	return rows
}

// FindUsersByIDs fetches name + avatar for the given user IDs.
func (r *RankingRepository) FindUsersByIDs(ids []int) []UserInfoRow {
	if len(ids) == 0 {
		return nil
	}
	var users []UserInfoRow
	r.db.Table(`"user"`).Select("id, name, avatar").
		Where("id IN ?", ids).Scan(&users)
	return users
}

// FindTopicRanking returns topic ranking rows with user info joined.
func (r *RankingRepository) FindTopicRanking(sortField, sortOrder string, page, limit int) []TopicRankingRow {
	var rows []TopicRankingRow
	r.db.Table("topic t").
		Select(`t.id, t.title, t.user_id, u.name AS user_name, u.avatar AS user_avatar,
			t.`+sortField+` AS value`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1").
		Order("t." + sortField + " " + sortOrder).
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return rows
}

// FindUserRanking returns user ranking rows.
func (r *RankingRepository) FindUserRanking(sortField, sortOrder string, page, limit int) []UserRankingRow {
	var rows []UserRankingRow
	r.db.Table(`"user"`).
		Select(`id, name, avatar, bio, ` + sortField + ` AS value`).
		Where("status != 1").
		Order(sortField + " " + sortOrder).
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return rows
}
