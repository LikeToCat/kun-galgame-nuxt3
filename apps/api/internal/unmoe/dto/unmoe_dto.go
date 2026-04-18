package dto

// ListRequest is the query for GET /unmoe.
type ListRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

// UnmoeUser is the slim author embed.
type UnmoeUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// UnmoeDescription is the four-language description map.
type UnmoeDescription struct {
	EnUs string `json:"en-us"`
	JaJp string `json:"ja-jp"`
	ZhCn string `json:"zh-cn"`
	ZhTw string `json:"zh-tw"`
}

// UnmoeLog is a single log entry returned by the list endpoint.
type UnmoeLog struct {
	ID          int              `json:"id"`
	User        UnmoeUser        `json:"user"`
	Name        string           `json:"name"`
	Description UnmoeDescription `json:"description"`
	Result      string           `json:"result"`
	Created     string           `json:"created"`
}

// ListResponse is the wrapped response for GET /unmoe.
type ListResponse struct {
	Logs  []UnmoeLog `json:"logs"`
	Total int64      `json:"total"`
}
