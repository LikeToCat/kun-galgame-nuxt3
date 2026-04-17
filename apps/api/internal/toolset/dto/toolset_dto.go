package dto

import (
	"encoding/json"

	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
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

// ToolsetDetailResponse is the response for GET /toolset/:id.
// Field names are lifted directly from the pre-refactor handler so the wire
// format is unchanged.
type ToolsetDetailResponse struct {
	Toolset         model.GalgameToolset             `json:"toolset"`
	DescriptionHTML string                           `json:"descriptionHTML"`
	Aliases         []model.GalgameToolsetAlias      `json:"aliases"`
	User            userModel.UserBrief              `json:"user"`
	Practicality    PracticalitySummary              `json:"practicality"`
	DownloadSum     int64                            `json:"downloadSum"`
	Comments        []CommentDetailItem              `json:"comments"`
	Contributors    []repository.ContributorBrief    `json:"contributors"`
	Resources       []model.GalgameToolsetResource   `json:"resources"`
}

// CreatedToolsetResponse is the raw toolset row returned by POST /toolset.
type CreatedToolsetResponse = model.GalgameToolset

// HomepageJSON is a convenience alias used by the service when encoding homepage.
type HomepageJSON = json.RawMessage
