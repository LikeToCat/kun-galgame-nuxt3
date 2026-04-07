package model

import "time"

// Report is a user-submitted report/complaint.
type Report struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Reason string `gorm:"type:varchar(1007);not null" json:"reason"`
	Type   string `gorm:"default:''" json:"type"`
	Status int    `gorm:"default:0" json:"status"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Report) TableName() string { return "report" }

// UpdateLog records system update changelogs with multi-language content.
type UpdateLog struct {
	ID            int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type          string `gorm:"not null" json:"type"`
	Version       string `gorm:"default:''" json:"version"`
	ContentEnUS   string `gorm:"column:content_en_us;type:text;default:''" json:"content_en_us"`
	ContentJaJP   string `gorm:"column:content_ja_jp;type:text;default:''" json:"content_ja_jp"`
	ContentZhCN   string `gorm:"column:content_zh_cn;type:text;default:''" json:"content_zh_cn"`
	ContentZhTW   string `gorm:"column:content_zh_tw;type:text;default:''" json:"content_zh_tw"`

	UserID int `gorm:"column:user_id;not null;default:2" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (UpdateLog) TableName() string { return "update_log" }

// Todo is an admin-only task item with multi-language content.
type Todo struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Type          string     `gorm:"default:'forum'" json:"type"` // forum, patch
	Status        int        `gorm:"default:0" json:"status"`
	ContentEnUS   string     `gorm:"column:content_en_us;type:text;default:''" json:"content_en_us"`
	ContentJaJP   string     `gorm:"column:content_ja_jp;type:text;default:''" json:"content_ja_jp"`
	ContentZhCN   string     `gorm:"column:content_zh_cn;type:text;default:''" json:"content_zh_cn"`
	ContentZhTW   string     `gorm:"column:content_zh_tw;type:text;default:''" json:"content_zh_tw"`
	CompletedTime *time.Time `gorm:"column:completed_time" json:"completed_time"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Todo) TableName() string { return "todo" }

// Unmoe is the unmoe character translator entries.
type Unmoe struct {
	ID         int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"not null" json:"name"`
	Result     string `gorm:"default:''" json:"result"`
	DescEnUS   string `gorm:"column:desc_en_us;type:text;default:''" json:"desc_en_us"`
	DescJaJP   string `gorm:"column:desc_ja_jp;type:text;default:''" json:"desc_ja_jp"`
	DescZhCN   string `gorm:"column:desc_zh_cn;type:text;default:''" json:"desc_zh_cn"`
	DescZhTW   string `gorm:"column:desc_zh_tw;type:text;default:''" json:"desc_zh_tw"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Unmoe) TableName() string { return "unmoe" }
