package dto

// This file holds parsing structs for Wiki Service responses.
// These types mirror the wire format produced by the wiki service;
// they are used by services when decoding `json.RawMessage` payloads.
//
// The fields are a superset — consumers ignore what they don't need.

// WikiAlias is a named alias entry used by Official/Tag/Galgame.
type WikiAlias struct {
	Name string `json:"name"`
}

// WikiGalgameItem is the shape returned inside list/detail responses of
// series/official/engine/tag endpoints (a "lite" galgame summary).
type WikiGalgameItem struct {
	ID                 int    `json:"id"`
	NameEnUs           string `json:"name_en_us"`
	NameJaJp           string `json:"name_ja_jp"`
	NameZhCn           string `json:"name_zh_cn"`
	NameZhTw           string `json:"name_zh_tw"`
	Banner             string `json:"banner"`
	ContentLimit       string `json:"content_limit"`
	View               int    `json:"view"`
	ResourceUpdateTime string `json:"resource_update_time"`
	UserID             int    `json:"user_id"`
}

// WikiOfficial is a company/publisher/developer entity from the wiki.
type WikiOfficial struct {
	ID       int         `json:"id"`
	Name     string      `json:"name"`
	Link     string      `json:"link"`
	Category string      `json:"category"`
	Lang     string      `json:"lang"`
	Alias    []WikiAlias `json:"alias"`
}

// WikiOfficialRel is the wrapper used when an official is attached to a galgame.
type WikiOfficialRel struct {
	Official WikiOfficial `json:"official"`
}

// WikiEngine is a game engine entity.
type WikiEngine struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Link  string `json:"link"`
	Intro string `json:"intro"`
}

// WikiEngineRel wraps an engine attached to a galgame.
type WikiEngineRel struct {
	Engine WikiEngine `json:"engine"`
}

// WikiTag is a tag entity with optional aliases.
type WikiTag struct {
	ID       int         `json:"id"`
	Name     string      `json:"name"`
	Category string      `json:"category"`
	Alias    []WikiAlias `json:"alias"`
}

// WikiTagRel wraps a tag attached to a galgame.
type WikiTagRel struct {
	Tag WikiTag `json:"tag"`
}

// WikiContributor represents a user who contributed to a galgame.
type WikiContributor struct {
	UserID int `json:"user_id"`
}

// WikiGalgameDetail is the core galgame payload returned by /galgame/:id.
// It contains the fields commonly consumed by the gateway service.
type WikiGalgameDetail struct {
	ID               int               `json:"id"`
	NameEnUs         string            `json:"name_en_us"`
	NameJaJp         string            `json:"name_ja_jp"`
	NameZhCn         string            `json:"name_zh_cn"`
	NameZhTw         string            `json:"name_zh_tw"`
	Banner           string            `json:"banner"`
	ContentLimit     string            `json:"content_limit"`
	AgeLimit         string            `json:"age_limit"`
	OriginalLanguage string            `json:"original_language"`
	Official         []WikiOfficialRel `json:"official"`
	Engine           []WikiEngineRel   `json:"engine"`
	Tag              []WikiTagRel      `json:"tag"`
	Contributors     []WikiContributor `json:"contributors"`
}

// WikiGalgameDetailResponse is the envelope: {galgame: {...}}.
type WikiGalgameDetailResponse struct {
	Galgame WikiGalgameDetail `json:"galgame"`
}

// WikiUser mirrors the user shape returned by wiki inside the `users` map of
// detail responses.
type WikiUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// WikiTagWithSpoiler extends WikiTag with a spoiler_level annotation used
// only in galgame detail responses.
type WikiTagWithSpoiler struct {
	SpoilerLevel int     `json:"spoiler_level"`
	Tag          WikiTag `json:"tag"`
}

// WikiEngineAlias is a flat alias slice (used in list/detail shape).
type WikiEngineAlias []string

// WikiGalgameDetailFull is the superset of fields returned by GET /galgame/:id.
// It includes nested alias/official/engine/tag/contributor/intro/user_id.
type WikiGalgameDetailFull struct {
	ID                 int                  `json:"id"`
	VndbID             string               `json:"vndb_id"`
	NameEnUs           string               `json:"name_en_us"`
	NameJaJp           string               `json:"name_ja_jp"`
	NameZhCn           string               `json:"name_zh_cn"`
	NameZhTw           string               `json:"name_zh_tw"`
	Banner             string               `json:"banner"`
	IntroEnUs          string               `json:"intro_en_us"`
	IntroJaJp          string               `json:"intro_ja_jp"`
	IntroZhCn          string               `json:"intro_zh_cn"`
	IntroZhTw          string               `json:"intro_zh_tw"`
	ContentLimit       string               `json:"content_limit"`
	View               int                  `json:"view"`
	ResourceUpdateTime string               `json:"resource_update_time"`
	OriginalLanguage   string               `json:"original_language"`
	AgeLimit           string               `json:"age_limit"`
	UserID             int                  `json:"user_id"`
	SeriesID           *int                 `json:"series_id"`
	Status             int                  `json:"status"`
	Alias              []WikiAlias          `json:"alias"`
	Official           []WikiOfficialRel    `json:"official"`
	Engine             []WikiEngineWithAlias `json:"engine"`
	Tag                []WikiTagWithSpoiler `json:"tag"`
	Contributor        []WikiContributor    `json:"contributor"`
	Created            string               `json:"created"`
	Updated            string               `json:"updated"`
}

// WikiEngineWithAlias matches the engine-embedded-in-galgame shape (alias is []string).
type WikiEngineWithAlias struct {
	Engine struct {
		ID    int      `json:"id"`
		Name  string   `json:"name"`
		Alias []string `json:"alias"`
	} `json:"engine"`
}

// WikiGalgameDetailFullResp is the envelope with galgame + users map.
type WikiGalgameDetailFullResp struct {
	Galgame WikiGalgameDetailFull `json:"galgame"`
	Users   map[string]WikiUser   `json:"users"`
}

// WikiSeriesSample is a sample galgame inside a series detail response.
type WikiSeriesSample struct {
	NameEnUs     string `json:"name_en_us"`
	NameJaJp     string `json:"name_ja_jp"`
	NameZhCn     string `json:"name_zh_cn"`
	NameZhTw     string `json:"name_zh_tw"`
	Banner       string `json:"banner"`
	ContentLimit string `json:"content_limit"`
}

// WikiSeriesBrief is the shape of /series/:id used inside GalgameDetail.
type WikiSeriesBrief struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Galgame     []WikiSeriesSample `json:"galgame"`
	Created     string             `json:"created"`
	Updated     string             `json:"updated"`
}

// WikiPRDetail is the shape returned by /galgame/:gid/prs/:id (inside "pr").
type WikiPRDetail struct {
	PR struct {
		UserID int `json:"user_id"`
	} `json:"pr"`
}

// WikiCreatedResp is the shape returned by POST /galgame (just the ID).
type WikiCreatedResp struct {
	ID int `json:"id"`
}
