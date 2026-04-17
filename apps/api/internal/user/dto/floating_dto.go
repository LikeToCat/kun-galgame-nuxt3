package dto

// FloatingCardRequest binds the query params for GET /api/user/:uid/floating.
// NOTE: the URL path parameter :uid is historically ignored — the frontend
// passes the target user via ?userId=N. Preserve this quirk for compatibility.
type FloatingCardRequest struct {
	UserID int `query:"userId" validate:"required,min=1"`
}

// FloatingCardResponse is the hover-card payload returned to the frontend.
type FloatingCardResponse struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	Avatar               string `json:"avatar"`
	Moemoepoint          int    `json:"moemoepoint"`
	TopicCount           int64  `json:"topicCount"`
	TopicReplyCount      int64  `json:"topicReplyCount"`
	TopicCommentCount    int64  `json:"topicCommentCount"`
	GalgameResourceCount int64  `json:"galgameResourceCount"`
}
