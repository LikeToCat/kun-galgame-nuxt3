package dto

import "time"

// ──────────────────────────────────────────
// Reply requests
// ──────────────────────────────────────────

type ListRepliesRequest struct {
	TopicID   int    `query:"topicId" validate:"required,min=1"`
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=30"`
	SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
}

type ReplyTarget struct {
	TargetReplyID int    `json:"targetReplyId" validate:"required,min=1"`
	Content       string `json:"content" validate:"max=10007"`
}

type CreateReplyRequest struct {
	TopicID int           `json:"topicId" validate:"required,min=1"`
	Content string        `json:"content" validate:"max=10007"`
	Targets []ReplyTarget `json:"targets" validate:"max=10"`
}

type UpdateReplyRequest struct {
	ReplyID int           `json:"replyId" validate:"required,min=1"`
	Content string        `json:"content" validate:"max=10007"`
	Targets []ReplyTarget `json:"targets" validate:"max=10"`
}

type ReplyInteractionRequest struct {
	ReplyID int `json:"replyId" validate:"required,min=1"`
}

type BestAnswerRequest struct {
	TopicID int `json:"topicId" validate:"required,min=1"`
	ReplyID int `json:"replyId" validate:"required,min=1"`
}

type PinReplyRequest struct {
	TopicID int `json:"topicId" validate:"required,min=1"`
	ReplyID int `json:"replyId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Reply responses
// ──────────────────────────────────────────

type TopicReplyResponse struct {
	ID              int                    `json:"id"`
	TopicID         int                    `json:"topicId"`
	Floor           int                    `json:"floor"`
	User            KunUserWithMoemoepoint `json:"user"`
	Edited          *time.Time             `json:"edited"`
	ContentMarkdown string                 `json:"contentMarkdown"`
	ContentHtml     string                 `json:"contentHtml"`
	LikeCount       int                    `json:"likeCount"`
	IsLiked         bool                   `json:"isLiked"`
	DislikeCount    int                    `json:"dislikeCount"`
	IsDisliked      bool                   `json:"isDisliked"`
	Comments        []TopicCommentResponse `json:"comment"`
	Targets         []ReplyTargetResponse  `json:"targets"`
	IsPinned        bool                   `json:"isPinned"`
	IsBestAnswer    bool                   `json:"isBestAnswer"`
	Created         time.Time              `json:"created"`
}

type ReplyTargetResponse struct {
	ID                   int     `json:"id"`
	Floor                int     `json:"floor"`
	User                 KunUser `json:"user"`
	ContentPreview       string  `json:"contentPreview"`
	ReplyContentMarkdown string  `json:"replyContentMarkdown"`
	ReplyContentHtml     string  `json:"replyContentHtml"`
}

// ──────────────────────────────────────────
// Comment requests
// ──────────────────────────────────────────

type CreateCommentRequest struct {
	TopicID      int    `json:"topicId" validate:"required,min=1"`
	ReplyID      int    `json:"replyId" validate:"required,min=1"`
	TargetUserID int    `json:"targetUserId" validate:"required,min=1"`
	Content      string `json:"content" validate:"required,min=1,max=1007"`
}

type CommentInteractionRequest struct {
	CommentID int `json:"commentId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Comment responses
// ──────────────────────────────────────────

type TopicCommentResponse struct {
	ID         int       `json:"id"`
	ReplyID    int       `json:"replyId"`
	TopicID    int       `json:"topicId"`
	User       KunUser   `json:"user"`
	TargetUser KunUser   `json:"targetUser"`
	Content    string    `json:"content"`
	IsLiked    bool      `json:"isLiked"`
	LikeCount  int       `json:"likeCount"`
	Created    time.Time `json:"created"`
}
