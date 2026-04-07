package dto

import "time"

// ──────────────────────────────────────────
// Auth
// ──────────────────────────────────────────

type OAuthCallbackRequest struct {
	Code         string `json:"code" validate:"required"`
	CodeVerifier string `json:"code_verifier" validate:"required"`
}

type SessionResponse struct {
	Token string       `json:"-"`
	User  *UserProfile `json:"user"`
}

type UserProfile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Avatar      string `json:"avatar"`
	Role        int    `json:"role"`
	Moemoepoint int    `json:"moemoepoint"`
	Bio         string `json:"bio"`
}

// ──────────────────────────────────────────
// User profile detail (GET /api/user/:uid)
// ──────────────────────────────────────────

type UserProfileDetail struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"`
	Role         int       `json:"role"`
	Status       int       `json:"status"`
	Moemoepoint  int       `json:"moemoepoint"`
	Bio          string    `json:"bio"`
	CreatedAt    time.Time `json:"created"`
	TopicCount   int64     `json:"topic_count"`
	ReplyCount   int64     `json:"reply_count"`
	GalgameCount int64     `json:"galgame_count"`
	LikeCount    int64     `json:"like_count"`
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
