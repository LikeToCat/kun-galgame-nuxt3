package dto

import "time"

// ──────────────────────────────────────────
// Auth
// ──────────────────────────────────────────

type OAuthCallbackRequest struct {
	Code         string `json:"code" validate:"required,max=2048"`
	CodeVerifier string `json:"code_verifier" validate:"required,max=256"`
}

type SessionResponse struct {
	Token string       `json:"-"`
	User  *UserProfile `json:"user"`
}

// UserProfile is the shape returned by /auth/oauth/callback and /auth/me.
// Email is OAuth-owned; the frontend fetches it via OAuth /oauth/userinfo
// when needed. Identity here is sourced from OAuth via pkg/userclient.
type UserProfile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Role        int    `json:"role"`
	Moemoepoint int    `json:"moemoepoint"`
	Bio         string `json:"bio"`
}

// ──────────────────────────────────────────
// User profile detail (GET /api/user/:uid)
// ──────────────────────────────────────────

type UserProfileDetail struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar"`
	Role        int       `json:"role"`
	Status      int       `json:"status"`
	Moemoepoint int       `json:"moemoepoint"`
	Bio         string    `json:"bio"`
	CreatedAt   time.Time `json:"created"`

	// Created counts
	Topic                  int64 `json:"topic"`
	TopicPoll              int64 `json:"topicPoll"`
	ReplyCreated           int64 `json:"replyCreated"`
	CommentCreated         int64 `json:"commentCreated"`
	Galgame                int64 `json:"galgame"`
	ContributeGalgame      int64 `json:"contributeGalgame"`
	GalgameComment         int64 `json:"galgameComment"`
	GalgameRating          int64 `json:"galgameRating"`
	GalgameResource        int64 `json:"galgameResource"`
	GalgameToolset         int64 `json:"galgameToolset"`
	GalgameToolsetResource int64 `json:"galgameToolsetResource"`

	// Received interaction counts
	Upvote  int64 `json:"upvote"`
	Like    int64 `json:"like"`
	Dislike int64 `json:"dislike"`

	// Daily counts
	DailyTopicCount   int64 `json:"dailyTopicCount"`
	DailyGalgameCount int64 `json:"dailyGalgameCount"`
}

// ──────────────────────────────────────────
// User mutations
// ──────────────────────────────────────────

type UpdateBioRequest struct {
	Bio string `json:"bio" validate:"max=107"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username" validate:"required,min=1,max=17"`
}

type UpdateEmailRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Code     string `json:"code" validate:"required"`
	CodeSalt string `json:"codeSalt" validate:"required"`
}

// SendResetCodeRequest is the body of POST /auth/email/code/reset — used by
// the "change email" flow. Email must NOT already belong to another user.
type SendResetCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ──────────────────────────────────────────
// User queries
// ──────────────────────────────────────────

type UserStatusResponse struct {
	Moemoepoints  int  `json:"moemoepoints"`
	IsCheckIn     bool `json:"isCheckIn"`
	HasNewMessage bool `json:"hasNewMessage"`
}

type UserGalgamesRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserTopicsRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserRepliesRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserCommentsRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserResourcesRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserRatingsRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type GalgameCard struct {
	ID               int       `json:"id"`
	VndbID           string    `json:"vndb_id"`
	NameEnUS         string    `json:"name_en_us"`
	NameJaJP         string    `json:"name_ja_jp"`
	NameZhCN         string    `json:"name_zh_cn"`
	NameZhTW         string    `json:"name_zh_tw"`
	Banner           string    `json:"banner"`
	ContentLimit     string    `json:"content_limit"`
	CreatedAt        time.Time `json:"created"`
}

type UserTopic struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created"`
}

// ──────────────────────────────────────────
// Admin
// ──────────────────────────────────────────

type BanUserRequest struct {
	Status int `json:"status" validate:"oneof=0 1"`
}
