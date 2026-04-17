package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type UpdateWebsiteTagRequest struct {
	TagID       int    `json:"tagId" validate:"required,min=1"`
	Name        string `json:"name" validate:"required"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

type DeleteWebsiteTagRequest struct {
	TagID int `query:"tagId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// WebsiteTagDetailResponse is the shape of GET /website-tag/:name.
type WebsiteTagDetailResponse struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Level        int           `json:"level"`
	Description  string        `json:"description"`
	WebsiteCount int           `json:"websiteCount"`
	Websites     []WebsiteCard `json:"websites"`
	Created      time.Time     `json:"created"`
	Updated      time.Time     `json:"updated"`
}
