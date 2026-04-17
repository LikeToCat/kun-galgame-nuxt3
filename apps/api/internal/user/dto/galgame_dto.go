package dto

// KunLanguage is the four-language locale map shared across galgame-related
// responses: { en-us, ja-jp, zh-cn, zh-tw }.
type KunLanguage struct {
	EnUs string `json:"en-us"`
	JaJp string `json:"ja-jp"`
	ZhCn string `json:"zh-cn"`
	ZhTw string `json:"zh-tw"`
}

// UserBrief is the minimal user shape embedded in list responses.
type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// UserGalgameCard is an item in GET /api/user/:uid/galgames.
type UserGalgameCard struct {
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
