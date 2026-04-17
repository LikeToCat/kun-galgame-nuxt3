package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// GetChatHistoryRequest is the query for GET /api/message/chat/history.
type GetChatHistoryRequest struct {
	ReceiverUID int `query:"receiverUid" validate:"required,min=1"`
	Page        int `query:"page" validate:"min=1"`
	Limit       int `query:"limit" validate:"min=1,max=50"`
}

// SendChatMessageRequest is the body for POST /api/message/chat/send.
type SendChatMessageRequest struct {
	ReceiverUID int    `json:"receiverUid" validate:"required,min=1"`
	Content     string `json:"content" validate:"required,min=1,max=1007"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// ChatSender is the sender object embedded in chat messages.
type ChatSender struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// ChatMessageItem is a single chat message item returned by GetChatHistory.
// Field names/shape must match the frontend exactly.
type ChatMessageItem struct {
	ID           int         `json:"id"`
	ChatroomName string      `json:"chatroomName"`
	Sender       ChatSender  `json:"sender"`
	ReceiverUID  int         `json:"receiverUid"`
	Content      string      `json:"content"`
	IsRecall     bool        `json:"isRecall"`
	Created      string      `json:"created"`
	RecallTime   *string     `json:"recallTime"`
	EditTime     *string     `json:"editTime"`
	ReadBy       []ChatSender `json:"readBy"`
}

// NavContactItem is a chat room entry for the message sidebar.
// Field names/shape must match the frontend exactly.
type NavContactItem struct {
	ChatroomName    string  `json:"chatroomName"`
	Content         string  `json:"content"`
	LastMessageTime *string `json:"lastMessageTime"`
	Count           int     `json:"count"`
	UnreadCount     int     `json:"unreadCount"`
	Route           string  `json:"route"`
	Title           string  `json:"title"`
	Avatar          string  `json:"avatar"`
}
