package model

import "time"

// WikiMessageReadState tracks how far each kungal user has consumed the
// wiki message stream (GET /galgame/messages/mine). Local to kungal: wiki
// itself doesn't store per-user read state because the same message can
// surface in multiple consumers (kungal / moyu / admin UI) with
// independent read state — see docs/galgame_wiki/08-messages.md §已读状态.
type WikiMessageReadState struct {
	UserID            int       `gorm:"column:user_id;primaryKey" json:"user_id"`
	LastReadMessageID int64     `gorm:"column:last_read_message_id;default:0" json:"last_read_message_id"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (WikiMessageReadState) TableName() string { return "wiki_message_read_state" }
