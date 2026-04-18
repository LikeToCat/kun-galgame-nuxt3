package dto

import "encoding/json"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type RatingListRequest struct {
	Page         int    `query:"page" validate:"min=1"`
	Limit        int    `query:"limit" validate:"min=1,max=50"`
	SortField    string `query:"sortField"`
	SortOrder    string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
	SpoilerLevel string `query:"spoilerLevel"`
	PlayStatus   string `query:"playStatus"`
	GalgameType  string `query:"galgameType"`
}

// CreateRatingRequest is the body of POST /galgame-rating.
// short_summary length drives the moemoepoint reward tier.
type CreateRatingRequest struct {
	GalgameID    int      `json:"galgameId" validate:"required,min=1"`
	Recommend    string   `json:"recommend" validate:"required"`
	Overall      int      `json:"overall" validate:"required,min=1,max=10"`
	GalgameType  []string `json:"galgameType" validate:"required,min=1"`
	PlayStatus   string   `json:"play_status" validate:"required"`
	ShortSummary string   `json:"short_summary"`
	SpoilerLevel string   `json:"spoiler_level"`
	Art          int      `json:"art" validate:"min=0,max=10"`
	Story        int      `json:"story" validate:"min=0,max=10"`
	Music        int      `json:"music" validate:"min=0,max=10"`
	Character    int      `json:"character" validate:"min=0,max=10"`
	Route        int      `json:"route" validate:"min=0,max=10"`
	System       int      `json:"system" validate:"min=0,max=10"`
	Voice        int      `json:"voice" validate:"min=0,max=10"`
	ReplayValue  int      `json:"replay_value" validate:"min=0,max=10"`
}

// UpdateRatingRequest is the body of PUT /galgame-rating/:id.
type UpdateRatingRequest struct {
	GalgameRatingID int      `json:"galgameRatingId" validate:"required,min=1"`
	Recommend       string   `json:"recommend" validate:"required"`
	Overall         int      `json:"overall" validate:"required,min=1,max=10"`
	GalgameType     []string `json:"galgameType" validate:"required,min=1"`
	PlayStatus      string   `json:"play_status" validate:"required"`
	ShortSummary    string   `json:"short_summary"`
	SpoilerLevel    string   `json:"spoiler_level" validate:"required"`
	Art             int      `json:"art" validate:"min=0,max=10"`
	Story           int      `json:"story" validate:"min=0,max=10"`
	Music           int      `json:"music" validate:"min=0,max=10"`
	Character       int      `json:"character" validate:"min=0,max=10"`
	Route           int      `json:"route" validate:"min=0,max=10"`
	System          int      `json:"system" validate:"min=0,max=10"`
	Voice           int      `json:"voice" validate:"min=0,max=10"`
	ReplayValue     int      `json:"replay_value" validate:"min=0,max=10"`
}

// DeleteRatingRequest is the query for DELETE /galgame-rating/:id.
type DeleteRatingRequest struct {
	GalgameRatingID int `query:"galgameRatingId" validate:"required,min=1"`
}

// ToggleRatingLikeRequest is the body of PUT /galgame-rating/:id/like.
type ToggleRatingLikeRequest struct {
	GalgameRatingID int `json:"galgameRatingId" validate:"required,min=1"`
}

// CreateRatingCommentRequest is the body of POST /galgame-rating/:id/comment.
type CreateRatingCommentRequest struct {
	GalgameRatingID int    `json:"galgameRatingId" validate:"required,min=1"`
	TargetUserID    int    `json:"targetUserId" validate:"required,min=1"`
	Content         string `json:"content" validate:"required,min=1,max=1314"`
}

// UpdateRatingCommentRequest is the body of PUT /galgame-rating/:id/comment.
type UpdateRatingCommentRequest struct {
	GalgameRatingCommentID int    `json:"galgameRatingCommentId" validate:"required,min=1"`
	Content                string `json:"content" validate:"required,min=1,max=1314"`
}

// DeleteRatingCommentRequest is the query for DELETE /galgame-rating/:id/comment.
type DeleteRatingCommentRequest struct {
	GalgameRatingCommentID int `query:"galgameRatingCommentId" validate:"required,min=1"`
}

// CreatedRating is the response shape for POST /galgame-rating — matches
// GalgameRatingCardOnGalgamePage in the frontend types.
type CreatedRating struct {
	ID           int             `json:"id"`
	User         UserBrief       `json:"user"`
	Recommend    string          `json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `json:"view"`
	GalgameType  json.RawMessage `json:"galgameType"`
	PlayStatus   string          `json:"play_status"`
	ShortSummary string          `json:"short_summary"`
	SpoilerLevel string          `json:"spoiler_level"`
	RatingScores                 // embedded scores
	LikeCount    int             `json:"likeCount"`
	IsLiked      bool            `json:"isLiked"`
	Created      string          `json:"created"`
	Updated      string          `json:"updated"`
	Galgame      RatingGalgameBrief `json:"galgame"`
}

// CreatedRatingComment is the response shape for POST/PUT comment.
type CreatedRatingComment struct {
	ID         int        `json:"id"`
	Content    string     `json:"content"`
	User       UserBrief  `json:"user"`
	TargetUser *UserBrief `json:"targetUser"`
	Created    string     `json:"created"`
	Updated    string     `json:"updated"`
}

// ──────────────────────────────────────────
// Shared scores embed
// ──────────────────────────────────────────

// RatingScores holds the per-axis rating scores.
// Spread into rating card + detail to match the frontend shape.
type RatingScores struct {
	Art         int `json:"art"`
	Story       int `json:"story"`
	Music       int `json:"music"`
	Character   int `json:"character"`
	Route       int `json:"route"`
	System      int `json:"system"`
	Voice       int `json:"voice"`
	ReplayValue int `json:"replay_value"`
}

// ──────────────────────────────────────────
// Responses — list
// ──────────────────────────────────────────

// RatingGalgameBrief is the lightweight galgame info in rating lists.
type RatingGalgameBrief struct {
	ID           int         `json:"id"`
	ContentLimit string      `json:"contentLimit"`
	Name         KunLanguage `json:"name"`
}

// RatingCard is a single entry in the rating list response.
type RatingCard struct {
	ID           int                `json:"id"`
	User         UserBrief          `json:"user"`
	Recommend    string             `json:"recommend"`
	Overall      int                `json:"overall"`
	View         int                `json:"view"`
	GalgameType  json.RawMessage    `json:"galgameType"`
	PlayStatus   string             `json:"play_status"`
	ShortSummary string             `json:"short_summary"`
	SpoilerLevel string             `json:"spoiler_level"`
	RatingScores                    // embedded fields art/story/music/...
	LikeCount    int                `json:"likeCount"`
	Created      string             `json:"created"`
	Updated      string             `json:"updated"`
	Galgame      RatingGalgameBrief `json:"galgame"`
}

type RatingListPage struct {
	RatingData []RatingCard `json:"ratingData"`
	Total      int64        `json:"total"`
}

// ──────────────────────────────────────────
// Responses — detail
// ──────────────────────────────────────────

// RatingOfficial is a single official/company entry shown in rating detail.
type RatingOfficial struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgameCount"`
}

// RatingGalgameDetail is the full galgame panel on the rating detail page.
type RatingGalgameDetail struct {
	ID               int              `json:"id"`
	ContentLimit     string           `json:"contentLimit"`
	Banner           string           `json:"banner"`
	AgeLimit         string           `json:"ageLimit"`
	OriginalLanguage string           `json:"originalLanguage"`
	Rating           int64            `json:"rating"`
	RatingCount      int64            `json:"ratingCount"`
	Official         []RatingOfficial `json:"official"`
	Name             KunLanguage      `json:"name"`
}

// RatingCommentItem is a reply on a rating.
type RatingCommentItem struct {
	ID         int        `json:"id"`
	Content    string     `json:"content"`
	User       UserBrief  `json:"user"`
	TargetUser *UserBrief `json:"targetUser"`
	Created    string     `json:"created"`
	Updated    string     `json:"updated"`
}

// RatingDetail is the full response for GET /galgame-rating/:id.
type RatingDetail struct {
	ID           int                 `json:"id"`
	User         UserBrief           `json:"user"`
	Recommend    string              `json:"recommend"`
	Overall      int                 `json:"overall"`
	View         int                 `json:"view"`
	GalgameType  json.RawMessage     `json:"galgameType"`
	PlayStatus   string              `json:"play_status"`
	ShortSummary string              `json:"short_summary"`
	SpoilerLevel string              `json:"spoiler_level"`
	RatingScores                     // embedded art/story/music/...
	LikeCount    int                 `json:"likeCount"`
	IsLiked      bool                `json:"isLiked"`
	LikedUsers   []UserBrief         `json:"likedUsers"`
	Comments     []RatingCommentItem `json:"comments"`
	Created      string              `json:"created"`
	Updated      string              `json:"updated"`
	Galgame      RatingGalgameDetail `json:"galgame"`
}
