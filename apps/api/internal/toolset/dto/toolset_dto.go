package dto

import (
	"encoding/json"
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type ToolsetListRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
	Type      string `query:"type"`
	Language  string `query:"language"`
	Platform  string `query:"platform"`
	Version   string `query:"version"`
	SortField string `query:"sortField"`
	SortOrder string `query:"sortOrder"`
}

type CreateToolsetRequest struct {
	Name        string   `json:"name" validate:"required,max=500"`
	Description string   `json:"description" validate:"max=2000"`
	Type        string   `json:"type"`
	Language    string   `json:"language"`
	Platform    string   `json:"platform"`
	Homepage    []string `json:"homepage"`
	Version     string   `json:"version" validate:"max=233"`
	Aliases     []string `json:"aliases"`
}

type UpdateToolsetRequest struct {
	Name        string   `json:"name" validate:"required,max=500"`
	Description string   `json:"description" validate:"max=2000"`
	Type        string   `json:"type"`
	Language    string   `json:"language"`
	Platform    string   `json:"platform"`
	Homepage    []string `json:"homepage"`
	Version     string   `json:"version" validate:"max=233"`
	Aliases     []string `json:"aliases"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// ToolsetCard is the shape of each item in the list response.
//
// Note: `PracticalityAvg` is `any` so we can emit `null` when there are no
// ratings (the original handler did the same).
type ToolsetCard struct {
	ID                 int                 `json:"id"`
	Name               string              `json:"name"`
	User               userModel.UserBrief `json:"user"`
	Type               string              `json:"type"`
	Platform           string              `json:"platform"`
	Language           string              `json:"language"`
	Version            string              `json:"version"`
	View               int                 `json:"view"`
	Download           int                 `json:"download"`
	CommentCount       int                 `json:"commentCount"`
	PracticalityAvg    any                 `json:"practicalityAvg"`
	ResourceUpdateTime any                 `json:"resource_update_time"`
}

// ToolsetResourceItem is the slim resource projection embedded in the toolset
// detail response. Hides sensitive fields (code/password/note) — those are
// only served by the dedicated /toolset/:id/resource/detail endpoint.
type ToolsetResourceItem struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Size     string `json:"size"`
	Download int    `json:"download"`
	Status   int    `json:"status"`
}

// ToolsetDetailResponse is the FLAT response for GET /toolset/:id, shaped to
// match the frontend `ToolsetDetail` type directly. Previously this was a
// nested envelope (`{toolset, descriptionHTML, ...}`) which made the
// frontend look up `data.toolset.created` — but the page reads `data.created`,
// causing `new Date(undefined).toISOString()` to throw RangeError.
type ToolsetDetailResponse struct {
	ID                 int                           `json:"id"`
	Name               string                        `json:"name"`
	ContentMarkdown    string                        `json:"contentMarkdown"`
	ContentHTML        string                        `json:"contentHtml"`
	Type               string                        `json:"type"`
	Platform           string                        `json:"platform"`
	Language           string                        `json:"language"`
	Version            string                        `json:"version"`
	Homepage           []string                      `json:"homepage"`
	View               int                           `json:"view"`
	Download           int64                         `json:"download"`
	User               userModel.UserBrief           `json:"user"`
	Aliases            []string                      `json:"aliases"`
	PracticalityAvg    *float64                      `json:"practicalityAvg"`
	PracticalityCount  int64                         `json:"practicalityCount"`
	RatingCounts       map[int]int64                 `json:"ratingCounts"`
	ResourceUpdateTime time.Time                     `json:"resource_update_time"`
	Resource           []ToolsetResourceItem         `json:"resource"`
	Edited             *time.Time                    `json:"edited"`
	Created            time.Time                     `json:"created"`
	Updated            time.Time                     `json:"updated"`
	CommentCount       int64                         `json:"commentCount"`
	CommentPreview     []CommentDetailItem           `json:"commentPreview"`
	Contributors       []userModel.UserBrief         `json:"contributors"`
}

// CreatedToolsetResponse is the raw toolset row returned by POST /toolset.
type CreatedToolsetResponse = model.GalgameToolset

// HomepageJSON is a convenience alias used by the service when encoding homepage.
type HomepageJSON = json.RawMessage
