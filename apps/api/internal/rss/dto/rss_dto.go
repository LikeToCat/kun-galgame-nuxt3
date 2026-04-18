package dto

import "time"

type TopicRSSItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      int       `json:"userId"`
	UserName    string    `json:"userName"`
	Created     time.Time `json:"created"`
}

// GalgameRSSUser is the slim author embed inside a galgame RSS item.
type GalgameRSSUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// GalgameRSSItem mirrors the legacy nitro RSSGalgame shape: a single
// best-effort name + intro, picked by language preference downstream.
type GalgameRSSItem struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Banner      string         `json:"banner"`
	User        GalgameRSSUser `json:"user"`
	Description string         `json:"description"`
	Created     string         `json:"created"`
}
