package model

// UserBrief is the (id, name, avatar) projection embedded into list/detail
// DTOs. Identity is owned by OAuth post-migration; this struct is populated
// by hydrating user_id values through pkg/userclient — never by reading from
// a local "user" table.
type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// UserStats is a projection for aggregated user statistics.
type UserStats struct {
	Topic                  int64 `gorm:"column:topic"`
	TopicPoll              int64 `gorm:"column:topic_poll"`
	ReplyCreated           int64 `gorm:"column:reply_created"`
	CommentCreated         int64 `gorm:"column:comment_created"`
	GalgameComment         int64 `gorm:"column:galgame_comment"`
	GalgameRating          int64 `gorm:"column:galgame_rating"`
	GalgameResource        int64 `gorm:"column:galgame_resource"`
	GalgameToolset         int64 `gorm:"column:galgame_toolset"`
	GalgameToolsetResource int64 `gorm:"column:galgame_toolset_resource"`
	Upvote                 int64 `gorm:"column:upvote"`
	Like                   int64 `gorm:"column:like"`
	Dislike                int64 `gorm:"column:dislike"`
	DailyTopicCount        int64 `gorm:"column:daily_topic_count"`
}
