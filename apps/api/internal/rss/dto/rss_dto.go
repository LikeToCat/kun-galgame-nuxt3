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
