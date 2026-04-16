package model

import "time"

// GalgameLocal represents the stripped-down local galgame row.
// After wiki migration, only interaction counts + view remain locally.
type GalgameLocal struct {
	ID               int `gorm:"primaryKey" json:"id"`
	View             int `gorm:"default:0" json:"view"`
	LikeCount        int `gorm:"column:like_count;default:0" json:"like_count"`
	FavoriteCount    int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	ResourceCount    int `gorm:"column:resource_count;default:0" json:"resource_count"`
	CommentCount     int `gorm:"column:comment_count;default:0" json:"comment_count"`
	ContributorCount int `gorm:"column:contributor_count;default:0" json:"contributor_count"`
	RatingCount      int `gorm:"column:rating_count;default:0" json:"rating_count"`
}

func (GalgameLocal) TableName() string { return "galgame" }

// ──────────────────────────────────────────
// Interactions (local to each site)
// ──────────────────────────────────────────

type GalgameLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_like" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameLike) TableName() string { return "galgame_like" }

type GalgameFavorite struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_favorite" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameFavorite) TableName() string { return "galgame_favorite" }

// ──────────────────────────────────────────
// Comment (local)
// ──────────────────────────────────────────

type GalgameComment struct {
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string `gorm:"type:varchar(1007);not null" json:"content"`
	GalgameID    int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID       int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID *int   `gorm:"column:target_user_id" json:"target_user_id"`
	LikeCount    int    `gorm:"column:like_count;default:0" json:"like_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameComment) TableName() string { return "galgame_comment" }

type GalgameCommentLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_comment_like" json:"user_id"`
	CommentID int `gorm:"column:galgame_comment_id;not null;uniqueIndex:idx_comment_like" json:"galgame_comment_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameCommentLike) TableName() string { return "galgame_comment_like" }
