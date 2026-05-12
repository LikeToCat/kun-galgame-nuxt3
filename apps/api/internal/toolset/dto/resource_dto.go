package dto

import (
	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// Wire field name is `toolsetResourceId` to match the frontend convention
// (validations/toolset.ts) and the legacy nitro endpoints. The internal Go
// field stays `ResourceID` since the service only deals with one kind of
// resource id at a time.

type ResourceDetailRequest struct {
	ResourceID int `query:"toolsetResourceId" validate:"required,min=1"`
}

type CreateResourceRequest struct {
	Content  string `json:"content" validate:"max=1007"`
	Type     string `json:"type" validate:"required,oneof=s3 user"`
	Code     string `json:"code" validate:"max=1007"`
	Password string `json:"password" validate:"max=1007"`
	Size     string `json:"size" validate:"max=107"`
	Note     string `json:"note" validate:"max=1007"`
}

type UpdateResourceRequest struct {
	ResourceID int    `json:"toolsetResourceId" validate:"required,min=1"`
	Content    string `json:"content" validate:"max=1007"`
	Code       string `json:"code" validate:"max=1007"`
	Password   string `json:"password" validate:"max=1007"`
	Size       string `json:"size" validate:"max=107"`
	Note       string `json:"note" validate:"max=1007"`
}

type DeleteResourceRequest struct {
	ResourceID int `query:"toolsetResourceId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// ResourceDetailResponse is returned by GET /toolset/:id/resource/detail.
// It embeds the raw resource model so the wire format matches the
// pre-refactor response exactly.
type ResourceDetailResponse struct {
	Resource model.GalgameToolsetResource `json:"resource"`
	User     userModel.UserBrief          `json:"user"`
}

// CreatedResourceResponse is the resource row returned by POST.
// (Handler returns the model directly; we preserve that.)
type CreatedResourceResponse = model.GalgameToolsetResource
