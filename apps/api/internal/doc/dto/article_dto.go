package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// GetArticlesRequest is the query for GET /doc/article.
type GetArticlesRequest struct {
	Page       int    `query:"page" validate:"min=1"`
	Limit      int    `query:"limit" validate:"min=1,max=50"`
	CategoryID *int   `query:"categoryId"`
	TagID      *int   `query:"tagId"`
	Status     *int   `query:"status"`
	IsPin      *bool  `query:"isPin"`
	Keyword    string `query:"keyword"`
	OrderBy    string `query:"orderBy" validate:"omitempty,oneof=published_time created view updated"`
	SortOrder  string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
}

// CreateArticleRequest is the payload for POST /doc/article.
type CreateArticleRequest struct {
	Title           string `json:"title" validate:"required,max=233"`
	Slug            string `json:"slug" validate:"required,max=233"`
	Description     string `json:"description" validate:"max=1000"`
	Banner          string `json:"banner" validate:"max=500"`
	Status          int    `json:"status" validate:"oneof=0 1 2"`
	IsPin           bool   `json:"isPin"`
	ContentMarkdown string `json:"contentMarkdown" validate:"required"`
	CategoryID      int    `json:"categoryId" validate:"required,min=1"`
	TagIDs          []int  `json:"tagIds"`
}

// UpdateArticleRequest is the payload for PUT /doc/article.
type UpdateArticleRequest struct {
	ArticleID       int    `json:"articleId" validate:"required,min=1"`
	Title           string `json:"title" validate:"required,max=233"`
	Slug            string `json:"slug" validate:"required,max=233"`
	Description     string `json:"description" validate:"max=1000"`
	Banner          string `json:"banner" validate:"max=500"`
	Status          int    `json:"status" validate:"oneof=0 1 2"`
	IsPin           bool   `json:"isPin"`
	ContentMarkdown string `json:"contentMarkdown" validate:"required"`
	CategoryID      int    `json:"categoryId" validate:"required,min=1"`
	TagIDs          []int  `json:"tagIds"`
}

// DeleteArticleRequest is the query for DELETE /doc/article.
type DeleteArticleRequest struct {
	ArticleID int `query:"articleId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// ArticleDetailResponse is the shape of GET /doc/article/:slug.
// Field names mirror the pre-refactor handler output exactly (mixing snake_case
// from the GORM model with one camelCase field for the rendered HTML).
type ArticleDetailResponse struct {
	ID              int        `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Path            string     `json:"path"`
	Description     string     `json:"description"`
	Banner          string     `json:"banner"`
	Status          int        `json:"status"`
	IsPin           bool       `json:"is_pin"`
	View            int        `json:"view"`
	PublishedTime   time.Time  `json:"published_time"`
	EditedTime      *time.Time `json:"edited_time"`
	ContentMarkdown string     `json:"content_markdown"`
	ContentHTML     string     `json:"contentHtml"`
	CategoryID      int        `json:"category_id"`
	AuthorID        int        `json:"author_id"`
	Created         time.Time  `json:"created"`
	Updated         time.Time  `json:"updated"`
}
