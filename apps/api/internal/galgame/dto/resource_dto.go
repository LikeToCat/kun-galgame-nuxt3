package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type ResourceListRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type GalgameResourcesRequest struct {
	GalgameID int `query:"galgameId" validate:"required,min=1"`
}

// CreateGalgameResourceRequest is the body of POST /galgame/:gid/resource.
type CreateGalgameResourceRequest struct {
	GalgameID int      `json:"galgameId" validate:"required,min=1"`
	Type      string   `json:"type" validate:"required"`
	Language  string   `json:"language" validate:"required"`
	Platform  string   `json:"platform" validate:"required"`
	Size      string   `json:"size" validate:"required,max=107"`
	Code      string   `json:"code" validate:"max=1007"`
	Password  string   `json:"password" validate:"max=1007"`
	Note      string   `json:"note" validate:"max=1007"`
	Link      []string `json:"link" validate:"required,min=1,max=20,dive,url"`
}

// UpdateGalgameResourceRequest is the body of PUT /galgame/:gid/resource.
type UpdateGalgameResourceRequest struct {
	GalgameResourceID int      `json:"galgameResourceId" validate:"required,min=1"`
	GalgameID         int      `json:"galgameId"`
	Type              string   `json:"type" validate:"required"`
	Language          string   `json:"language" validate:"required"`
	Platform          string   `json:"platform" validate:"required"`
	Size              string   `json:"size" validate:"required,max=107"`
	Code              string   `json:"code" validate:"max=1007"`
	Password          string   `json:"password" validate:"max=1007"`
	Note              string   `json:"note" validate:"max=1007"`
	Link              []string `json:"link" validate:"required,min=1,max=20,dive,url"`
}

// DeleteGalgameResourceRequest is the query for DELETE /galgame/:gid/resource.
type DeleteGalgameResourceRequest struct {
	GalgameResourceID int `query:"galgameResourceId" validate:"required,min=1"`
}

// ToggleResourceLikeRequest is the body of PUT /galgame/:gid/resource/like.
type ToggleResourceLikeRequest struct {
	GalgameResourceID int `json:"galgameResourceId" validate:"required,min=1"`
}

// ResourceStatusRequest is the body of PUT valid/expired endpoints.
type ResourceStatusRequest struct {
	GalgameResourceID int `json:"galgameResourceId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// UserBrief is a lightweight user projection used in resource responses.
type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// KunLanguage is a four-language text map.
type KunLanguage struct {
	EnUs string `json:"en-us"`
	JaJp string `json:"ja-jp"`
	ZhCn string `json:"zh-cn"`
	ZhTw string `json:"zh-tw"`
}

// ResourceCard is the shape returned in list views (no links/code/password).
type ResourceCard struct {
	ID          int         `json:"id"`
	View        int         `json:"view"`
	GalgameID   int         `json:"galgameId"`
	User        UserBrief   `json:"user"`
	Type        string      `json:"type"`
	Language    string      `json:"language"`
	Platform    string      `json:"platform"`
	Size        string      `json:"size"`
	Status      int         `json:"status"`
	Download    int         `json:"download"`
	LikeCount   int         `json:"likeCount"`
	IsLiked     bool        `json:"isLiked"`
	LinkDomain  string      `json:"linkDomain"`
	Note        string      `json:"note"`
	Created     string      `json:"created"`
	Edited      *string     `json:"edited"`
	GalgameName KunLanguage `json:"galgameName,omitempty"`
}

// ResourceDownloadDetail is returned by GET /galgame-resource/:id/detail.
// Includes download links, code, password, note.
type ResourceDownloadDetail struct {
	ID         int       `json:"id"`
	View       int       `json:"view"`
	GalgameID  int       `json:"galgameId"`
	User       UserBrief `json:"user"`
	Type       string    `json:"type"`
	Language   string    `json:"language"`
	Platform   string    `json:"platform"`
	Size       string    `json:"size"`
	Status     int       `json:"status"`
	Download   int       `json:"download"`
	LikeCount  int       `json:"likeCount"`
	IsLiked    bool      `json:"isLiked"`
	LinkDomain string    `json:"linkDomain"`
	Link       []string  `json:"link"`
	Code       string    `json:"code"`
	Password   string    `json:"password"`
	Note       string    `json:"note"`
	Created    string    `json:"created"`
	Edited     *string   `json:"edited"`
}

// ResourceGalgameSummary is the galgame info shown on resource detail page.
type ResourceGalgameSummary struct {
	ID                 int         `json:"id"`
	Name               KunLanguage `json:"name"`
	Banner             string      `json:"banner"`
	ContentLimit       string      `json:"contentLimit"`
	View               int         `json:"view"`
	ResourceUpdateTime string      `json:"resourceUpdateTime"`
	OriginalLanguage   string      `json:"originalLanguage"`
	AgeLimit           string      `json:"ageLimit"`
	Platform           []string    `json:"platform"`
	Language           []string    `json:"language"`
	Type               []string    `json:"type"`
}

// ResourceDetailPage is the full response for GET /galgame-resource/:id.
type ResourceDetailPage struct {
	Galgame         ResourceGalgameSummary `json:"galgame"`
	Resource        ResourceDownloadDetail `json:"resource"`
	Recommendations []ResourceCard         `json:"recommendations"`
}

// ResourceListPage is the response for GET /galgame-resource.
type ResourceListPage struct {
	Resources []ResourceCard `json:"resources"`
	Total     int64          `json:"total"`
}
