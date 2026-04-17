package model

// GalgameRatingRow is a flat projection of galgame_rating for read queries.
type GalgameRatingRow struct {
	ID           int    `gorm:"column:id"`
	Recommend    string `gorm:"column:recommend"`
	Overall      int    `gorm:"column:overall"`
	View         int    `gorm:"column:view"`
	GalgameType  string `gorm:"column:galgame_type"`
	PlayStatus   string `gorm:"column:play_status"`
	ShortSummary string `gorm:"column:short_summary"`
	SpoilerLevel string `gorm:"column:spoiler_level"`
	Art          int    `gorm:"column:art"`
	Story        int    `gorm:"column:story"`
	Music        int    `gorm:"column:music"`
	Character    int    `gorm:"column:character"`
	Route        int    `gorm:"column:route"`
	System       int    `gorm:"column:system"`
	Voice        int    `gorm:"column:voice"`
	ReplayValue  int    `gorm:"column:replay_value"`
	LikeCount    int    `gorm:"column:like_count"`
	Created      string `gorm:"column:created"`
	Updated      string `gorm:"column:updated"`
	UserID       int    `gorm:"column:user_id"`
	GalgameID    int    `gorm:"column:galgame_id"`
}

// RatingCommentRow is a joined comment row for a rating.
type RatingCommentRow struct {
	ID           int    `gorm:"column:id"`
	Content      string `gorm:"column:content"`
	UserID       int    `gorm:"column:user_id"`
	TargetUserID *int   `gorm:"column:target_user_id"`
	UserName     string `gorm:"column:user_name"`
	UserAvatar   string `gorm:"column:user_avatar"`
	TargetName   string `gorm:"column:target_name"`
	TargetAvatar string `gorm:"column:target_avatar"`
	Created      string `gorm:"column:created"`
	Updated      string `gorm:"column:updated"`
}

// RatingFilter carries the list-query filters to the repository.
type RatingFilter struct {
	SpoilerLevel string
	PlayStatus   string
	GalgameType  string
	SortField    string
	SortOrder    string
	Page         int
	Limit        int
}
