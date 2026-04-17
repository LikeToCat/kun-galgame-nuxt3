package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// GetStatsRequest is the query for GET /admin/overview/stats.
type GetStatsRequest struct {
	Days int `query:"days" validate:"min=1,max=365"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// OverviewItem is one row of GET /admin/overview/all.
type OverviewItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// DailyStatRow is one day of GET /admin/overview/stats. Each row contains a
// `date` field plus one field per model key (user, topic, …). It is stored as
// an ordered key/value map to preserve JSON emission order and match the
// pre-refactor frontend shape exactly.
type DailyStatRow map[string]any
