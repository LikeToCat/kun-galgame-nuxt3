package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// GetCategoriesRequest is the query for GET /doc/category.
type GetCategoriesRequest struct {
	Page    int    `query:"page" validate:"min=1"`
	Limit   int    `query:"limit" validate:"min=1,max=50"`
	Keyword string `query:"keyword"`
}

// UpdateCategoryRequest is the payload for PUT /doc/category.
type UpdateCategoryRequest struct {
	CategoryID  int    `json:"categoryId" validate:"required,min=1"`
	Slug        string `json:"slug" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sortOrder"`
}

// DeleteCategoryRequest is the query for DELETE /doc/category.
type DeleteCategoryRequest struct {
	CategoryID int `query:"categoryId" validate:"required,min=1"`
}
