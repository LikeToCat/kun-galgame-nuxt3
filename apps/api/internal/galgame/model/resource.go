package model

import "time"

// GalgameResource is the writable model for the galgame_resource table,
// used for inserts and field-level updates. The `provider` text[] column
// is maintained via raw SQL (GORM has no native array support).
type GalgameResource struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string `gorm:"type:varchar" json:"type"`
	Language  string `gorm:"type:varchar" json:"language"`
	Platform  string `gorm:"type:varchar" json:"platform"`
	Size      string `gorm:"type:varchar(107)" json:"size"`
	Code      string `gorm:"type:varchar(1007)" json:"code"`
	Password  string `gorm:"type:varchar(1007)" json:"password"`
	Note      string `gorm:"type:varchar(1007)" json:"note"`
	View      int    `gorm:"default:0" json:"view"`
	Status    int    `gorm:"default:0" json:"status"`
	Download  int    `gorm:"default:0" json:"download"`
	GalgameID int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID    int    `gorm:"column:user_id;not null" json:"user_id"`
	LikeCount int    `gorm:"column:like_count;default:0" json:"like_count"`

	Edited    *time.Time `gorm:"column:edited" json:"edited"`
	CreatedAt time.Time  `gorm:"column:created" json:"created"`
	UpdatedAt time.Time  `gorm:"column:updated" json:"updated"`
}

func (GalgameResource) TableName() string { return "galgame_resource" }

// GalgameResourceLink mirrors the galgame_resource_link association row.
type GalgameResourceLink struct {
	ID                int    `gorm:"primaryKey;autoIncrement" json:"id"`
	URL               string `gorm:"column:url" json:"url"`
	GalgameResourceID int    `gorm:"column:galgame_resource_id;not null" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLink) TableName() string { return "galgame_resource_link" }

// GalgameResourceLike tracks which user liked which resource.
type GalgameResourceLike struct {
	ID                int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int `gorm:"column:user_id;not null;uniqueIndex:idx_resource_like" json:"user_id"`
	GalgameResourceID int `gorm:"column:galgame_resource_id;not null;uniqueIndex:idx_resource_like" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLike) TableName() string { return "galgame_resource_like" }

// GalgameResourceRow is a flat projection of galgame_resource used for reads.
// It doesn't drive migrations; it's the shape that repository queries return.
type GalgameResourceRow struct {
	ID        int     `gorm:"column:id"`
	View      int     `gorm:"column:view"`
	GalgameID int     `gorm:"column:galgame_id"`
	UserID    int     `gorm:"column:user_id"`
	Type      string  `gorm:"column:type"`
	Language  string  `gorm:"column:language"`
	Platform  string  `gorm:"column:platform"`
	Size      string  `gorm:"column:size"`
	Status    int     `gorm:"column:status"`
	Download  int     `gorm:"column:download"`
	LikeCount int     `gorm:"column:like_count"`
	Code      string  `gorm:"column:code"`
	Password  string  `gorm:"column:password"`
	Note      string  `gorm:"column:note"`
	Created   string  `gorm:"column:created"`
	Edited    *string `gorm:"column:edited"`
}

// ResourceAggregate is used when aggregating DISTINCT platform/language/type
// per galgame.
type ResourceAggregate struct {
	Platform string `gorm:"column:platform"`
	Language string `gorm:"column:language"`
	Type     string `gorm:"column:type"`
}
