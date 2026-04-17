package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type ActivityRequest struct {
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
	Type  string `query:"type" validate:"required"`
}

type TimelineRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

type Actor struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type ActivityItem struct {
	UniqueID  string    `json:"uniqueId"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     Actor     `json:"actor"`
	Link      string    `json:"link"`
	Content   string    `json:"content"`
}
