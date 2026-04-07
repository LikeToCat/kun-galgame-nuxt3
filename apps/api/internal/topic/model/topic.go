package model

import "time"

// ──────────────────────────────────────────
// Topic core
// ──────────────────────────────────────────

type Topic struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"type:varchar(233);not null" json:"title"`
	Content          string     `gorm:"type:text;not null" json:"content"`
	View             int        `gorm:"default:0" json:"view"`
	IsNSFW           bool       `gorm:"column:is_nsfw;default:false" json:"is_nsfw"`
	Status           int        `gorm:"default:0" json:"status"` // 0=normal, 1=banned, 2=pinned, 3=essential, 4=locked
	Category         string     `gorm:"not null" json:"category"`
	StatusUpdateTime time.Time  `gorm:"column:status_update_time;autoCreateTime" json:"status_update_time"`
	Edited           *time.Time `gorm:"" json:"edited"`
	UpvoteTime       *time.Time `gorm:"column:upvote_time" json:"upvote_time"`

	UserID        int  `gorm:"column:user_id;not null" json:"user_id"`
	BestAnswerID  *int `gorm:"column:best_answer_id;uniqueIndex" json:"best_answer_id"`
	PinnedReplyID *int `gorm:"column:pinned_reply_id;uniqueIndex" json:"pinned_reply_id"`

	// Counts (denormalized)
	LikeCount     int `gorm:"column:like_count;default:0" json:"like_count"`
	DislikeCount  int `gorm:"column:dislike_count;default:0" json:"dislike_count"`
	ReplyCount    int `gorm:"column:reply_count;default:0" json:"reply_count"`
	CommentCount  int `gorm:"column:comment_count;default:0" json:"comment_count"`
	FavoriteCount int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	UpvoteCount   int `gorm:"column:upvote_count;default:0" json:"upvote_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Topic) TableName() string { return "topic" }

// ──────────────────────────────────────────
// Tag (replaces topic.tag text[])
// ──────────────────────────────────────────

type TopicTag struct {
	ID   int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"uniqueIndex;not null" json:"name"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicTag) TableName() string { return "topic_tag" }

type TopicTagRelation struct {
	TopicID int `gorm:"column:topic_id;primaryKey" json:"topic_id"`
	TagID   int `gorm:"column:tag_id;primaryKey" json:"tag_id"`

	Tag TopicTag `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE" json:"tag,omitzero"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicTagRelation) TableName() string { return "topic_tag_relation" }

// ──────────────────────────────────────────
// Section
// ──────────────────────────────────────────

type TopicSection struct {
	ID   int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"not null" json:"name"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicSection) TableName() string { return "topic_section" }

type TopicSectionRelation struct {
	TopicID        int `gorm:"column:topic_id;primaryKey" json:"topic_id"`
	TopicSectionID int `gorm:"column:topic_section_id;primaryKey" json:"topic_section_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicSectionRelation) TableName() string { return "topic_section_relation" }

// ──────────────────────────────────────────
// Interactions
// ──────────────────────────────────────────

type TopicLike struct {
	ID      int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID int `gorm:"column:topic_id;not null;uniqueIndex:idx_topic_like" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null;uniqueIndex:idx_topic_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicLike) TableName() string { return "topic_like" }

type TopicDislike struct {
	ID      int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID int `gorm:"column:topic_id;not null;uniqueIndex:idx_topic_dislike" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null;uniqueIndex:idx_topic_dislike" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicDislike) TableName() string { return "topic_dislike" }

// TopicUpvote allows duplicate upvotes (no unique constraint).
type TopicUpvote struct {
	ID      int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID int `gorm:"column:topic_id;not null" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicUpvote) TableName() string { return "topic_upvote" }

type TopicFavorite struct {
	ID      int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID int `gorm:"column:topic_id;not null;uniqueIndex:idx_topic_favorite" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null;uniqueIndex:idx_topic_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicFavorite) TableName() string { return "topic_favorite" }

// ──────────────────────────────────────────
// Reply
// ──────────────────────────────────────────

type TopicReply struct {
	ID      int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string     `gorm:"type:text;default:''" json:"content"`
	Floor   int        `gorm:"default:0" json:"floor"`
	Edited  *time.Time `gorm:"" json:"edited"`

	UserID  int `gorm:"column:user_id;not null" json:"user_id"`
	TopicID int `gorm:"column:topic_id;not null" json:"topic_id"`

	// Counts (denormalized)
	LikeCount    int `gorm:"column:like_count;default:0" json:"like_count"`
	DislikeCount int `gorm:"column:dislike_count;default:0" json:"dislike_count"`
	CommentCount int `gorm:"column:comment_count;default:0" json:"comment_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReply) TableName() string { return "topic_reply" }

type TopicReplyLike struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int `gorm:"column:user_id;not null;uniqueIndex:idx_reply_like" json:"user_id"`
	TopicReplyID int `gorm:"column:topic_reply_id;not null;uniqueIndex:idx_reply_like" json:"topic_reply_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReplyLike) TableName() string { return "topic_reply_like" }

type TopicReplyDislike struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int `gorm:"column:user_id;not null;uniqueIndex:idx_reply_dislike" json:"user_id"`
	TopicReplyID int `gorm:"column:topic_reply_id;not null;uniqueIndex:idx_reply_dislike" json:"topic_reply_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReplyDislike) TableName() string { return "topic_reply_dislike" }

// TopicReplyTarget records when a reply targets another reply.
type TopicReplyTarget struct {
	ID            int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content       string `gorm:"type:text;default:''" json:"content"`
	ReplyID       int    `gorm:"column:reply_id;not null;uniqueIndex:idx_reply_target" json:"reply_id"`
	TargetReplyID int    `gorm:"column:target_reply_id;not null;uniqueIndex:idx_reply_target" json:"target_reply_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReplyTarget) TableName() string { return "topic_reply_target" }

// ──────────────────────────────────────────
// Comment (on replies)
// ──────────────────────────────────────────

type TopicComment struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string `gorm:"type:varchar(1007);default:''" json:"content"`
	TopicID      int    `gorm:"column:topic_id;not null" json:"topic_id"`
	TopicReplyID int    `gorm:"column:topic_reply_id;not null" json:"topic_reply_id"`
	UserID       int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID int    `gorm:"column:target_user_id;not null" json:"target_user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicComment) TableName() string { return "topic_comment" }

type TopicCommentLike struct {
	ID             int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicCommentID int `gorm:"column:topic_comment_id;not null;uniqueIndex:idx_comment_like" json:"topic_comment_id"`
	UserID         int `gorm:"column:user_id;not null;uniqueIndex:idx_comment_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicCommentLike) TableName() string { return "topic_comment_like" }

// ──────────────────────────────────────────
// Poll
// ──────────────────────────────────────────

type TopicPoll struct {
	ID          int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string     `gorm:"type:varchar(100);not null" json:"title"`
	Description string     `gorm:"type:varchar(500);default:''" json:"description"`
	Type        string     `gorm:"default:'single'" json:"type"`     // single, multiple
	MinChoice   int        `gorm:"column:min_choice;default:1" json:"min_choice"`
	MaxChoice   int        `gorm:"column:max_choice;default:1" json:"max_choice"`
	Deadline    *time.Time `gorm:"" json:"deadline"`
	Status      string     `gorm:"default:'open'" json:"status"`     // open, closed
	NotificationSent bool `gorm:"column:notification_sent;default:false" json:"notification_sent"`
	ResultVisibility string `gorm:"column:result_visibility;default:'always'" json:"result_visibility"` // always, after_vote, after_deadline
	IsAnonymous      bool   `gorm:"column:is_anonymous;default:false" json:"is_anonymous"`
	CanChangeVote    bool   `gorm:"column:can_change_vote;default:true" json:"can_change_vote"`

	TopicID int `gorm:"column:topic_id;not null" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPoll) TableName() string { return "topic_poll" }

type TopicPollOption struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Text   string `gorm:"type:varchar(100);not null" json:"text"`
	PollID int    `gorm:"column:poll_id;not null" json:"poll_id"`

	VoteCount int `gorm:"column:vote_count;default:0" json:"vote_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPollOption) TableName() string { return "topic_poll_option" }

type TopicPollVote struct {
	ID       int `gorm:"primaryKey;autoIncrement" json:"id"`
	PollID   int `gorm:"column:poll_id;not null;uniqueIndex:idx_poll_vote" json:"poll_id"`
	OptionID int `gorm:"column:option_id;not null;uniqueIndex:idx_poll_vote" json:"option_id"`
	UserID   int `gorm:"column:user_id;not null;uniqueIndex:idx_poll_vote;index:idx_user_poll" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPollVote) TableName() string { return "topic_poll_vote" }
