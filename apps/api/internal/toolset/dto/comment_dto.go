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
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=100"`
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
