package model

import "time"

type DocCategory struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Slug        string `gorm:"uniqueIndex;type:varchar(128);not null" json:"slug"`
	Title       string `gorm:"type:varchar(233);not null" json:"title"`
	Description string `gorm:"type:varchar(777);default:''" json:"description"`
	Icon        string `gorm:"type:varchar(128);default:''" json:"icon"`
	SortOrder   int    `gorm:"column:sort_order;default:0" json:"sort_order"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (DocCategory) TableName() string { return "doc_category" }

type DocTag struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Slug        string `gorm:"uniqueIndex;type:varchar(128);not null" json:"slug"`
	Title       string `gorm:"type:varchar(128);not null" json:"title"`
	Description string `gorm:"type:varchar(255);default:''" json:"description"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (DocTag) TableName() string { return "doc_tag" }

type DocArticle struct {
	ID              int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string     `gorm:"type:varchar(233);not null" json:"title"`
	Slug            string     `gorm:"uniqueIndex;type:varchar(233);not null" json:"slug"`
	Path            string     `gorm:"uniqueIndex;type:varchar(255);not null" json:"path"`
	Description     string     `gorm:"type:varchar(777);not null" json:"description"`
	Banner          string     `gorm:"type:varchar(777);default:''" json:"banner"`
	Status          int        `gorm:"default:1" json:"status"` // 0=draft, 1=published, 2=archived
	IsPin           bool       `gorm:"column:is_pin;default:false" json:"is_pin"`
	View            int        `gorm:"default:0" json:"view"`
	PublishedTime   time.Time  `gorm:"column:published_time;autoCreateTime" json:"published_time"`
	EditedTime      *time.Time `gorm:"column:edited_time" json:"edited_time"`
	ContentMarkdown string     `gorm:"column:content_markdown;type:varchar(100007);not null" json:"content_markdown"`

	CategoryID int `gorm:"column:category_id;not null" json:"category_id"`
	AuthorID   int `gorm:"column:author_id;not null" json:"author_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (DocArticle) TableName() string { return "doc_article" }

type DocArticleTagRelation struct {
	DocArticleID int `gorm:"column:doc_article_id;primaryKey" json:"doc_article_id"`
	DocTagID     int `gorm:"column:doc_tag_id;primaryKey" json:"doc_tag_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (DocArticleTagRelation) TableName() string { return "doc_article_tag_relation" }
