package dto

import "time"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type ListMessagesRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=30"`
	Type      string `query:"type"`
	SortOrder string `query:"sortOrder" validate:"required,oneof=asc desc"`
}

type ChatHistoryRequest struct {
	RoomID    string `query:"roomId" validate:"required"`
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=50"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

type KunUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type MessageResponse struct {
	ID          int       `json:"id"`
	Sender      KunUser   `json:"sender"`
	ReceiverUID int       `json:"receiverUid"`
	Link        string    `json:"link"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Created     time.Time `json:"created"`
}

type MessageListResponse struct {
	Messages   []MessageResponse `json:"messages"`
	TotalCount int64             `json:"totalCount"`
}

type SystemMessageResponse struct {
	ID      int     `json:"id"`
	Status  string  `json:"status"`
	Content map[string]string `json:"content"`
	Admin   KunUser `json:"admin"`
	Created time.Time `json:"created"`
}

type NavItem struct {
	Type          string  `json:"type"`
	TotalCount    int64   `json:"totalCount"`
	UnreadCount   int64   `json:"unreadCount"`
	LatestContent string  `json:"latestContent"`
	LatestTime    *time.Time `json:"latestTime"`
}
