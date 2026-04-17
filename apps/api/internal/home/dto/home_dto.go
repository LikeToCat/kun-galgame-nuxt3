package dto

import "time"

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

type LocaleName struct {
	EnUS string `json:"en-us"`
	JaJP string `json:"ja-jp"`
	ZhCN string `json:"zh-cn"`
	ZhTW string `json:"zh-tw"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type HomeGalgame struct {
	ID                 int        `json:"id"`
	Name               LocaleName `json:"name"`
	Banner             string     `json:"banner"`
	User               UserBrief  `json:"user"`
	ContentLimit       string     `json:"contentLimit"`
	View               int        `json:"view"`
	LikeCount          int        `json:"likeCount"`
	ResourceUpdateTime string     `json:"resourceUpdateTime"`
	Platform           []string   `json:"platform"`
	Language           []string   `json:"language"`
}

type HomeTopic struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	View             int        `json:"view"`
	LikeCount        int        `json:"likeCount"`
	ReplyCount       int        `json:"replyCount"`
	CommentCount     int        `json:"commentCount"`
	HasBestAnswer    bool       `json:"hasBestAnswer"`
	IsPollTopic      bool       `json:"isPollTopic"`
	IsNSFWTopic      bool       `json:"isNSFWTopic"`
	Section          []string   `json:"section"`
	Tag              []string   `json:"tag"`
	User             UserBrief  `json:"user"`
	Status           int        `json:"status"`
	UpvoteTime       *time.Time `json:"upvoteTime"`
	StatusUpdateTime time.Time  `json:"statusUpdateTime"`
}

type HomeResponse struct {
	Galgames []HomeGalgame `json:"galgames"`
	Topics   []HomeTopic   `json:"topics"`
}
