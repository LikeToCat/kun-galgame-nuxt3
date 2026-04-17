package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type CommentListRequest struct {
	WebsiteID int `query:"websiteId" validate:"required,min=1"`
}

type CreateCommentRequest struct {
	Content   string `json:"content" validate:"required,min=1,max=1007"`
	WebsiteID int    `json:"websiteId" validate:"required,min=1"`
	ParentID  *int   `json:"parentId"`
}

type DeleteCommentRequest struct {
	CommentID int `query:"commentId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// CommentUser is the user projection embedded in a comment.
type CommentUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// CommentItem is a single nested comment returned by GET /website/:domain/comment.
// Field names (camelCase) mirror the pre-refactor handler exactly.
type CommentItem struct {
	ID         int            `json:"id"`
	Content    string         `json:"content"`
	ParentID   *int           `json:"parentId"`
	UserID     int            `json:"userId"`
	WebsiteID  int            `json:"websiteId"`
	Created    string         `json:"created"`
	Edited     *string        `json:"edited"`
	Reply      []*CommentItem `json:"reply"`
	User       CommentUser    `json:"user"`
	TargetUser any            `json:"targetUser"`
}
