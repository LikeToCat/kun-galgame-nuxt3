package dto

// UserResourceItem is an entry in GET /api/user/:uid/resources.
type UserResourceItem struct {
	ID          int         `json:"id"`
	GalgameID   int         `json:"galgameId"`
	GalgameName KunLanguage `json:"galgameName"`
	Type        string      `json:"type"`
	Language    string      `json:"language"`
	Platform    string      `json:"platform"`
	Size        string      `json:"size"`
	Link        []string    `json:"link"`
	Code        string      `json:"code"`
	Password    string      `json:"password"`
	Note        string      `json:"note"`
	Status      int         `json:"status"`
	Created     string      `json:"created"`
}

// UserResourcesResponse is the payload for GET /api/user/:uid/resources.
type UserResourcesResponse struct {
	Resources []UserResourceItem `json:"resources"`
	Total     int64              `json:"total"`
}
