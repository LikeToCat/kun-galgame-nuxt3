package dto

// UserRatingGalgame is the embedded galgame metadata on a rating.
type UserRatingGalgame struct {
	ID           int         `json:"id"`
	Name         KunLanguage `json:"name"`
	ContentLimit string      `json:"contentLimit"`
}

// UserRatingItem mirrors the legacy rating-list response shape. Field names
// are intentionally a mix of camelCase and snake_case to match the existing
// frontend contract — do NOT rename without also updating the client.
type UserRatingItem struct {
	ID           int               `json:"id"`
	User         UserBrief         `json:"user"`
	Recommend    string            `json:"recommend"`
	Overall      int               `json:"overall"`
	View         int               `json:"view"`
	GalgameType  []string          `json:"galgameType"`
	PlayStatus   string            `json:"play_status"`
	Art          int               `json:"art"`
	Story        int               `json:"story"`
	Music        int               `json:"music"`
	Character    int               `json:"character"`
	Route        int               `json:"route"`
	System       int               `json:"system"`
	Voice        int               `json:"voice"`
	ReplayValue  int               `json:"replay_value"`
	SpoilerLevel string            `json:"spoiler_level"`
	LikeCount    int               `json:"likeCount"`
	Created      string            `json:"created"`
	Updated      string            `json:"updated"`
	Galgame      UserRatingGalgame `json:"galgame"`
}

// UserRatingsResponse is the payload for GET /api/user/:uid/ratings.
type UserRatingsResponse struct {
	RatingData []UserRatingItem `json:"ratingData"`
	Total      int64            `json:"total"`
}
