package dto

import "time"

// ──────────────────────────────────────────
// Shared user projections
// ──────────────────────────────────────────

type KunUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type KunUserWithMoemoepoint struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Moemoepoint int    `json:"moemoepoint"`
}

// ──────────────────────────────────────────
// Topic list
// ──────────────────────────────────────────

type ListTopicsRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=30"`
	SortField string `query:"sortField" validate:"required"`
	SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
	Category  string `query:"category" validate:"required"`
}

type TopicCard struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	View             int        `json:"view"`
	Tags             []string   `json:"tag"`
	Sections         []string   `json:"section"`
	User             KunUser    `json:"user"`
	Status           int        `json:"status"`
	HasBestAnswer    bool       `json:"hasBestAnswer"`
	IsPollTopic      bool       `json:"isPollTopic"`
	IsNSFW           bool       `json:"isNSFWTopic"`
	LikeCount        int        `json:"likeCount"`
	ReplyCount       int        `json:"replyCount"`
	CommentCount     int        `json:"commentCount"`
	StatusUpdateTime time.Time  `json:"statusUpdateTime"`
	UpvoteTime       *time.Time `json:"upvoteTime"`
}

// ──────────────────────────────────────────
// Topic detail
// ──────────────────────────────────────────

type TopicDetail struct {
	ID               int                    `json:"id"`
	Title            string                 `json:"title"`
	Content          string                 `json:"contentMarkdown"`
	ContentHtml      string                 `json:"contentHtml"`
	View             int                    `json:"view"`
	Status           int                    `json:"status"`
	IsNSFW           bool                   `json:"isNSFW"`
	Category         string                 `json:"category"`
	Sections         []string               `json:"section"`
	Tags             []string               `json:"tag"`
	User             KunUserWithMoemoepoint `json:"user"`
	LikeCount        int                    `json:"likeCount"`
	IsLiked          bool                   `json:"isLiked"`
	DislikeCount     int                    `json:"dislikeCount"`
	IsDisliked       bool                   `json:"isDisliked"`
	FavoriteCount    int                    `json:"favoriteCount"`
	IsFavorited      bool                   `json:"isFavorited"`
	UpvoteCount      int                    `json:"upvoteCount"`
	IsUpvoted        bool                   `json:"isUpvoted"`
	ReplyCount       int                    `json:"replyCount"`
	IsPollTopic      bool                   `json:"isPollTopic"`
	StatusUpdateTime time.Time              `json:"statusUpdateTime"`
	UpvoteTime       *time.Time             `json:"upvoteTime"`
	Edited           *time.Time             `json:"edited"`
	Created          time.Time              `json:"created"`
}

// ──────────────────────────────────────────
// Topic mutations
// ──────────────────────────────────────────

type CreateTopicRequest struct {
	Title    string   `json:"title" validate:"required,min=1,max=233"`
	Content  string   `json:"content" validate:"required,min=1,max=100007"`
	Tags     []string `json:"tag" validate:"required,min=1,max=7"`
	Category string   `json:"category" validate:"required,oneof=galgame technique others"`
	Sections []string `json:"section" validate:"required,min=1,max=3"`
	IsNSFW   bool     `json:"is_nsfw"`
}

type UpdateTopicRequest struct {
	Title    string   `json:"title" validate:"required,min=1,max=233"`
	Content  string   `json:"content" validate:"required,min=1,max=100007"`
	Tags     []string `json:"tag" validate:"required,min=1,max=7"`
	Category string   `json:"category" validate:"required,oneof=galgame technique others"`
	Sections []string `json:"section" validate:"required,min=1,max=3"`
	IsNSFW   bool     `json:"is_nsfw"`
}

type TopicInteractionRequest struct {
	TopicID int `json:"topicId" validate:"required,min=1"`
}
