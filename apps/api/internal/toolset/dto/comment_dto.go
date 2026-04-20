package dto

import (
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type CommentListRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
	SortOrder string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
}

// ToolsetCommentItem is the camelCase shape returned by GET
// /toolset/:id/comment/all. Mirrors the legacy nitro `ToolsetComment`
// frontend type one-for-one (parentId null when top-level, targetUser nil
// when not a reply). The `reply` field is intentionally always [] — the
// frontend renders the tree from the flat list itself.
type ToolsetCommentItem struct {
	ID         int                  `json:"id"`
	ToolsetID  int                  `json:"toolsetId"`
	Content    string               `json:"content"`
	Created    time.Time            `json:"created"`
	Edited     *time.Time           `json:"edited"`
	ParentID   *int                 `json:"parentId"`
	UserID     int                  `json:"userId"`
	Reply      []ToolsetCommentItem `json:"reply"`
	User       userModel.UserBrief  `json:"user"`
	TargetUser *userModel.UserBrief `json:"targetUser"`
}

// ToolsetCommentListResponse is the wrapper for GET /toolset/:id/comment/all.
// Uses `commentData` + `total` to match the frontend Container.vue contract.
type ToolsetCommentListResponse struct {
	CommentData []ToolsetCommentItem `json:"commentData"`
	Total       int64                `json:"total"`
}

type CreateCommentRequest struct {
	Content  string `json:"content" validate:"required,min=1,max=1007"`
	ParentID *int   `json:"parentId"`
}

type UpdateCommentRequest struct {
	CommentID int    `json:"commentId" validate:"required,min=1"`
	Content   string `json:"content" validate:"required,min=1,max=1007"`
}

type DeleteCommentRequest struct {
	CommentID int `query:"commentId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// CommentItem is the shape returned by GET /toolset/:id/comment.
// It embeds the raw GalgameToolsetComment model so the wire format is
// unchanged from the pre-refactor response.
type CommentItem struct {
	model.GalgameToolsetComment
	User       userModel.UserBrief  `json:"user"`
	ParentUser *userModel.UserBrief `json:"parent_user,omitempty"`
}

// CommentDetailItem is a slim comment + user projection used by the toolset
// detail response (no ParentUser field).
type CommentDetailItem struct {
	model.GalgameToolsetComment
	User userModel.UserBrief `json:"user"`
}

// CreatedCommentResponse mirrors the raw comment row returned by POST.
// (The original handler returned the model directly; we preserve that.)
type CreatedCommentResponse = model.GalgameToolsetComment

// UpdatedCommentResponse carries the fields the UpdateComment service modifies.
type UpdatedCommentResponse struct {
	Content string     `json:"content"`
	Edited  *time.Time `json:"edited"`
}
