package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type ListQuery struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type CreateHistoryRequest struct {
	Type        string `json:"type" validate:"required"`
	Version     string `json:"version"`
	ContentEnUS string `json:"content_en_us"`
	ContentJaJP string `json:"content_ja_jp"`
	ContentZhCN string `json:"content_zh_cn"`
	ContentZhTW string `json:"content_zh_tw"`
}

type DeleteHistoryRequest struct {
	ID int `query:"updateLogId" validate:"required,min=1"`
}

type CreateTodoRequest struct {
	Type        string `json:"type" validate:"required"`
	Status      int    `json:"status"`
	ContentEnUS string `json:"content_en_us"`
	ContentJaJP string `json:"content_ja_jp"`
	ContentZhCN string `json:"content_zh_cn"`
	ContentZhTW string `json:"content_zh_tw"`
}

type DeleteTodoRequest struct {
	ID int `query:"todoId" validate:"required,min=1"`
}
