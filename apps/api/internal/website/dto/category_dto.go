package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type UpdateWebsiteCategoryRequest struct {
	CategoryID  int    `json:"categoryId" validate:"required,min=1"`
	Name        string `json:"name" validate:"required"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// WebsiteCategoryDetailResponse is the shape of GET /website-category/:name.
// Field names mirror the pre-refactor handler output exactly.
type WebsiteCategoryDetailResponse struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Description  string        `json:"description"`
	WebsiteCount int           `json:"websiteCount"`
	Websites     []WebsiteCard `json:"websites"`
	Created      time.Time     `json:"created"`
	Updated      time.Time     `json:"updated"`
}
