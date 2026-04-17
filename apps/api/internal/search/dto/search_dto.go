package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type SearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
	Type     string `query:"type" validate:"required,oneof=topic galgame user reply comment"`
	Page     int    `query:"page" validate:"min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=12"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type TopicItem struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	View             int       `json:"view"`
	Status           int       `json:"status"`
	LikeCount        int       `json:"likeCount"`
	ReplyCount       int       `json:"replyCount"`
	CommentCount     int       `json:"commentCount"`
	StatusUpdateTime time.Time `json:"statusUpdateTime"`
	User             UserBrief `json:"user"`
}

type UserItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar"`
	Bio         string    `json:"bio"`
	Moemoepoint int       `json:"moemoepoint"`
	Created     time.Time `json:"created"`
}

type ReplyItem struct {
	ID         int       `json:"id"`
	TopicID    int       `json:"topicId"`
	TopicTitle string    `json:"topicTitle"`
	Content    string    `json:"content"`
	Floor      int       `json:"floor"`
	User       UserBrief `json:"user"`
	Created    time.Time `json:"created"`
}

type CommentItem struct {
	ID         int       `json:"id"`
	TopicID    int       `json:"topicId"`
	TopicTitle string    `json:"topicTitle"`
	Content    string    `json:"content"`
	User       UserBrief `json:"user"`
	Created    time.Time `json:"created"`
}

// PaginatedResult is a generic paginated payload returned by service methods.
type PaginatedResult[T any] struct {
	Items []T
	Total int64
}
