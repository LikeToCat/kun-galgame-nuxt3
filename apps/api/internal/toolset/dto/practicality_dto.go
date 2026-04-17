package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type UpsertPracticalityRequest struct {
	Rate int `json:"rate" validate:"required,min=1,max=5"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// PracticalityResponse is the payload returned by GET /toolset/:id/practicality.
// The Mine field is nil when the current user has not rated or is anonymous.
type PracticalityResponse struct {
	Counts map[int]int64 `json:"counts"`
	Avg    float64       `json:"avg"`
	Mine   *int          `json:"mine"`
}

// PracticalitySummary is the (counts, avg) pair embedded in the toolset detail
// response. It has no "mine" field.
type PracticalitySummary struct {
	Counts map[int]int64 `json:"counts"`
	Avg    float64       `json:"avg"`
}
