package model

import (
	"encoding/json"
	"time"
)

// ──────────────────────────────────────────
// Toolset core
// ──────────────────────────────────────────

type GalgameToolset struct {
	ID                 int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string          `gorm:"type:varchar(500);default:''" json:"name"`
	Description        string          `gorm:"type:varchar(2000);default:''" json:"description"`
	Status             int             `gorm:"default:0" json:"status"`
	View               int             `gorm:"default:0" json:"view"`
	Type               string          `gorm:"default:''" json:"type"`
	Language           string          `gorm:"default:''" json:"language"`
	Platform           string          `gorm:"default:''" json:"platform"`
	Homepage           json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"homepage"` // text[] → jsonb
	ResourceUpdateTime time.Time       `gorm:"column:resource_update_time;autoCreateTime" json:"resource_update_time"`
	Edited             *time.Time      `gorm:"" json:"edited"`
	Version            string          `gorm:"type:varchar(233);default:''" json:"version"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolset) TableName() string { return "galgame_toolset" }

// ──────────────────────────────────────────
// Contributor & Practicality
// ──────────────────────────────────────────

type GalgameToolsetContributor struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	ToolsetID int `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_contributor" json:"toolset_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_toolset_contributor" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetContributor) TableName() string { return "galgame_toolset_contributor" }

type GalgameToolsetPracticality struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	Rate      int `gorm:"default:1" json:"rate"` // 1-10
	UserID    int `gorm:"column:user_id;not null" json:"user_id"`
	ToolsetID int `gorm:"column:toolset_id;not null" json:"toolset_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetPracticality) TableName() string { return "galgame_toolset_practicality" }

// ──────────────────────────────────────────
// Alias
// ──────────────────────────────────────────

type GalgameToolsetAlias struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"default:''" json:"name"`
	ToolsetID int    `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_alias" json:"toolset_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetAlias) TableName() string { return "galgame_toolset_alias" }

// ──────────────────────────────────────────
// Resource
// ──────────────────────────────────────────

type GalgameToolsetResource struct {
	ID        int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Content   string     `gorm:"type:varchar(1007);default:''" json:"content"`
	Type      string     `gorm:"default:''" json:"type"` // s3, user
	Code      string     `gorm:"type:varchar(1007);default:''" json:"code"`
	Password  string     `gorm:"type:varchar(1007);default:''" json:"password"`
	Size      string     `gorm:"type:varchar(107);default:''" json:"size"`
	Note      string     `gorm:"type:varchar(1007);default:''" json:"note"`
	Download  int        `gorm:"default:0" json:"download"`
	Status    int        `gorm:"default:0" json:"status"`
	Edited    *time.Time `gorm:"" json:"edited"`

	ToolsetID int `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_resource" json:"toolset_id"`
	UserID    int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetResource) TableName() string { return "galgame_toolset_resource" }

// ──────────────────────────────────────────
// Category
// ──────────────────────────────────────────

type GalgameToolsetCategory struct {
	ID          int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"uniqueIndex;not null" json:"name"`
	Description string          `gorm:"default:''" json:"description"`
	Alias       json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"alias"` // text[] → jsonb

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetCategory) TableName() string { return "galgame_toolset_category" }

type GalgameToolsetCategoryRelation struct {
	ToolsetID  int `gorm:"column:toolset_id;primaryKey" json:"toolset_id"`
	CategoryID int `gorm:"column:category_id;primaryKey" json:"category_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetCategoryRelation) TableName() string { return "galgame_toolset_category_relation" }

// ──────────────────────────────────────────
// Comment (self-referencing tree)
// ──────────────────────────────────────────

type GalgameToolsetComment struct {
	ID        int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Content   string     `gorm:"default:''" json:"content"`
	Edited    *time.Time `gorm:"" json:"edited"`
	UserID    int        `gorm:"column:user_id;not null;index" json:"user_id"`
	ToolsetID int        `gorm:"column:toolset_id;not null;index" json:"toolset_id"`
	ParentID  *int       `gorm:"column:parent_id" json:"parent_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetComment) TableName() string { return "galgame_toolset_comment" }
