package dto

import "encoding/json"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type GalgameListRequest struct {
	Page                 int    `query:"page" validate:"min=1"`
	Limit                int    `query:"limit" validate:"min=1,max=50"`
	Type                 string `query:"type"`
	Language             string `query:"language"`
	Platform             string `query:"platform"`
	SortField            string `query:"sortField"`
	SortOrder            string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
	IncludeProviders     string `query:"includeProviders"`
	ExcludeOnlyProviders string `query:"excludeOnlyProviders"`
}

// ──────────────────────────────────────────
// Response: list
// ──────────────────────────────────────────

// GalgameListCard matches the existing frontend card used on galgame listings.
// Note: platform/language are denormalised from galgame_resource.
type GalgameListCard struct {
	ID                 int         `json:"id"`
	Name               KunLanguage `json:"name"`
	Banner             string      `json:"banner"`
	User               UserBrief   `json:"user"`
	ContentLimit       string      `json:"contentLimit"`
	View               int         `json:"view"`
	LikeCount          int         `json:"likeCount"`
	ResourceUpdateTime string      `json:"resourceUpdateTime"`
	Platform           []string    `json:"platform"`
	Language           []string    `json:"language"`
}

// GalgameListPage is the {galgames, total} envelope for GET /galgame.
type GalgameListPage struct {
	Galgames []GalgameListCard `json:"galgames"`
	Total    int64             `json:"total"`
}

// ──────────────────────────────────────────
// Response: detail
// ──────────────────────────────────────────

// GalgameDetailOfficial is an official entry on the detail page.
type GalgameDetailOfficial struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgameCount"`
}

// GalgameDetailEngine is an engine entry on the detail page.
type GalgameDetailEngine struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgameCount"`
}

// GalgameDetailTag is a tag entry on the detail page (with spoiler_level).
type GalgameDetailTag struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgameCount"`
	SpoilerLevel int    `json:"spoilerLevel"`
}

// GalgameDetailSeries is the series info shown on the detail page.
type GalgameDetailSeries struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsNSFW        bool            `json:"isNSFW"`
	SampleGalgame []GalgameSample `json:"sampleGalgame"`
	GalgameCount  int             `json:"galgameCount"`
	Created       string          `json:"created"`
	Updated       string          `json:"updated"`
}

// GalgameDetailRatingGalgame is the tiny galgame embed inside each rating card.
type GalgameDetailRatingGalgame struct {
	ID           int         `json:"id"`
	ContentLimit string      `json:"contentLimit"`
	Name         KunLanguage `json:"name"`
}

// GalgameDetailRating is a rating shown on the galgame detail page.
type GalgameDetailRating struct {
	ID           int                        `json:"id"`
	User         UserBrief                  `json:"user"`
	Recommend    string                     `json:"recommend"`
	Overall      int                        `json:"overall"`
	View         int                        `json:"view"`
	GalgameType  json.RawMessage            `json:"galgameType"`
	PlayStatus   string                     `json:"play_status"`
	ShortSummary string                     `json:"short_summary"`
	SpoilerLevel string                     `json:"spoiler_level"`
	Art          int                        `json:"art"`
	Story        int                        `json:"story"`
	Music        int                        `json:"music"`
	Character    int                        `json:"character"`
	Route        int                        `json:"route"`
	System       int                        `json:"system"`
	Voice        int                        `json:"voice"`
	ReplayValue  int                        `json:"replay_value"`
	LikeCount    int                        `json:"likeCount"`
	IsLiked      bool                       `json:"isLiked"`
	GalgameID    int                        `json:"galgameId"`
	Created      string                     `json:"created"`
	Updated      string                     `json:"updated"`
	Galgame      GalgameDetailRatingGalgame `json:"galgame"`
}

// GalgameDetail is the full response for GET /galgame/:gid.
type GalgameDetail struct {
	ID                 int                     `json:"id"`
	VndbID             string                  `json:"vndbId"`
	User               UserBrief               `json:"user"`
	Name               KunLanguage             `json:"name"`
	Banner             string                  `json:"banner"`
	Introduction       KunLanguage             `json:"introduction"`
	Markdown           KunLanguage             `json:"markdown"`
	ContentLimit       string                  `json:"contentLimit"`
	ResourceUpdateTime string                  `json:"resourceUpdateTime"`
	View               int                     `json:"view"`
	OriginalLanguage   string                  `json:"originalLanguage"`
	AgeLimit           string                  `json:"ageLimit"`
	Platform           []string                `json:"platform"`
	Language           []string                `json:"language"`
	Type               []string                `json:"type"`
	Contributor        []UserBrief             `json:"contributor"`
	LikeCount          int                     `json:"likeCount"`
	IsLiked            bool                    `json:"isLiked"`
	FavoriteCount      int                     `json:"favoriteCount"`
	IsFavorited        bool                    `json:"isFavorited"`
	Alias              []string                `json:"alias"`
	Series             *GalgameDetailSeries    `json:"series"`
	Engine             []GalgameDetailEngine   `json:"engine"`
	Official           []GalgameDetailOfficial `json:"official"`
	Tag                []GalgameDetailTag      `json:"tag"`
	Ratings            []GalgameDetailRating   `json:"ratings"`
	Created            string                  `json:"created"`
	Updated            string                  `json:"updated"`
}
