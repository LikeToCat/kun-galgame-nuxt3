package model

import (
	"encoding/json"
	"time"
)

// GalgameRating is the writable model for the galgame_rating table.
// galgame_type is JSONB; we wrap json.RawMessage to pass it through verbatim.
type GalgameRating struct {
	ID           int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Recommend    string          `gorm:"type:varchar" json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `gorm:"default:0" json:"view"`
	GalgameType  json.RawMessage `gorm:"column:galgame_type;type:jsonb" json:"galgame_type"`
	PlayStatus   string          `gorm:"column:play_status;type:varchar" json:"play_status"`
	ShortSummary string          `gorm:"column:short_summary;type:varchar(1314)" json:"short_summary"`
	SpoilerLevel string          `gorm:"column:spoiler_level;type:varchar" json:"spoiler_level"`
	Art          int             `gorm:"default:0" json:"art"`
	Story        int             `gorm:"default:0" json:"story"`
	Music        int             `gorm:"default:0" json:"music"`
	Character    int             `gorm:"default:0" json:"character"`
	Route        int             `gorm:"default:0" json:"route"`
	System       int             `gorm:"default:0" json:"system"`
	Voice        int             `gorm:"default:0" json:"voice"`
	ReplayValue  int             `gorm:"column:replay_value;default:0" json:"replay_value"`
	GalgameID    int             `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID       int             `gorm:"column:user_id;not null" json:"user_id"`
	LikeCount    int             `gorm:"column:like_count;default:0" json:"like_count"`
	CommentCount int             `gorm:"column:comment_count;default:0" json:"comment_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRating) TableName() string { return "galgame_rating" }

// GalgameRatingLike tracks rating likes (one row per user per rating).
type GalgameRatingLike struct {
	ID              int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameRatingID int `gorm:"column:galgame_rating_id;not null;uniqueIndex:idx_rating_like" json:"galgame_rating_id"`
	UserID          int `gorm:"column:user_id;not null;uniqueIndex:idx_rating_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRatingLike) TableName() string { return "galgame_rating_like" }

// GalgameRatingComment is the writable model for rating-comment rows.
type GalgameRatingComment struct {
	ID              int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content         string `gorm:"type:varchar(1314)" json:"content"`
	GalgameRatingID int    `gorm:"column:galgame_rating_id;not null" json:"galgame_rating_id"`
	UserID          int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID    *int   `gorm:"column:target_user_id" json:"target_user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRatingComment) TableName() string { return "galgame_rating_comment" }

// GalgameRatingRow is a flat projection of galgame_rating for read queries.
type GalgameRatingRow struct {
	ID           int    `gorm:"column:id"`
	Recommend    string `gorm:"column:recommend"`
	Overall      int    `gorm:"column:overall"`
	View         int    `gorm:"column:view"`
	GalgameType  string `gorm:"column:galgame_type"`
	PlayStatus   string `gorm:"column:play_status"`
	ShortSummary string `gorm:"column:short_summary"`
	SpoilerLevel string `gorm:"column:spoiler_level"`
	Art          int    `gorm:"column:art"`
	Story        int    `gorm:"column:story"`
	Music        int    `gorm:"column:music"`
	Character    int    `gorm:"column:character"`
	Route        int    `gorm:"column:route"`
	System       int    `gorm:"column:system"`
	Voice        int    `gorm:"column:voice"`
	ReplayValue  int    `gorm:"column:replay_value"`
	LikeCount    int    `gorm:"column:like_count"`
	Created      string `gorm:"column:created"`
	Updated      string `gorm:"column:updated"`
	UserID       int    `gorm:"column:user_id"`
	GalgameID    int    `gorm:"column:galgame_id"`
}

// RatingCommentRow is a comment row for a rating. Identity is hydrated by
// the service layer via userclient.
type RatingCommentRow struct {
	ID           int    `gorm:"column:id"`
	Content      string `gorm:"column:content"`
	UserID       int    `gorm:"column:user_id"`
	TargetUserID *int   `gorm:"column:target_user_id"`
	Created      string `gorm:"column:created"`
	Updated      string `gorm:"column:updated"`
}

// RatingFilter carries the list-query filters to the repository.
type RatingFilter struct {
	SpoilerLevel string
	PlayStatus   string
	GalgameType  string
	SortField    string
	SortOrder    string
	Page         int
	Limit        int
}
