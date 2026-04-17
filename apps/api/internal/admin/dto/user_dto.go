package dto

import "time"

// GetUserListRequest is the query for GET /admin/user.
type GetUserListRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=100"`
}

// AdminUserRow is one row of the admin user listing/search.
// Mirrors the pre-refactor handler output exactly.
type AdminUserRow struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Avatar  string    `json:"avatar"`
	Status  int       `json:"status"`
	Created time.Time `json:"created"`
}

// UserListResponse is the shape of GET /admin/user.
type UserListResponse struct {
	Users      []AdminUserRow `json:"users"`
	TotalCount int64          `json:"totalCount"`
}
