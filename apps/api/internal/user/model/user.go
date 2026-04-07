package model

import "time"

type User struct {
	ID                     int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                   string    `gorm:"uniqueIndex;not null" json:"name"`
	Email                  string    `gorm:"uniqueIndex;not null" json:"email"`
	Password               string    `gorm:"not null" json:"-"`
	IP                     string    `gorm:"default:''" json:"-"`
	Avatar                 string    `gorm:"default:''" json:"avatar"`
	Role                   int       `gorm:"default:1" json:"role"`
	Status                 int       `gorm:"default:0" json:"status"` // 0=normal, 1=banned
	Moemoepoint            int       `gorm:"default:7" json:"moemoepoint"`
	Bio                    string    `gorm:"type:varchar(107);default:''" json:"bio"`
	DailyCheckIn           int       `gorm:"column:daily_check_in;default:0" json:"-"`
	DailyImageCount        int       `gorm:"column:daily_image_count;default:0" json:"-"`
	DailyToolsetUploadCount int      `gorm:"column:daily_toolset_upload_count;default:0" json:"-"`
	CreatedAt              time.Time `gorm:"column:created" json:"created"`
	UpdatedAt              time.Time `gorm:"column:updated" json:"updated"`
}

func (User) TableName() string { return "user" }

// UserBrief is a lightweight projection for includes/preloads.
type UserBrief struct {
	ID     int    `gorm:"primaryKey" json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (UserBrief) TableName() string { return "user" }

// OAuthAccount links an OAuth provider sub (UUID) to a local user.
type OAuthAccount struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int       `gorm:"column:user_id;not null;index" json:"user_id"`
	Provider  string    `gorm:"not null;default:'kun-oauth'" json:"provider"` // "kun-oauth"
	Sub       string    `gorm:"not null;uniqueIndex" json:"sub"`             // OAuth UUID
	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (OAuthAccount) TableName() string { return "oauth_account" }

// UserFollow represents a follow relationship between users.
type UserFollow struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"id"`
	FollowerID int       `gorm:"column:follower_id;not null" json:"follower_id"`
	FollowedID int       `gorm:"column:followed_id;not null" json:"followed_id"`
	CreatedAt  time.Time `gorm:"column:created" json:"created"`
	UpdatedAt  time.Time `gorm:"column:updated" json:"updated"`

	Follower User `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE" json:"-"`
	Followed User `gorm:"foreignKey:FollowedID;constraint:OnDelete:CASCADE" json:"-"`
}

func (UserFollow) TableName() string { return "user_follow" }

// UserFriend represents a bidirectional friend relationship.
type UserFriend struct {
	ID       int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   int       `gorm:"column:user_id;not null" json:"user_id"`
	FriendID int       `gorm:"column:friend_id;not null" json:"friend_id"`
	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`

	User   User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Friend User `gorm:"foreignKey:FriendID;constraint:OnDelete:CASCADE" json:"-"`
}

func (UserFriend) TableName() string { return "user_friend" }

// UserStats is a projection for aggregated user statistics.
type UserStats struct {
	TopicCount   int64 `gorm:"column:topic_count" json:"topic_count"`
	ReplyCount   int64 `gorm:"column:reply_count" json:"reply_count"`
	GalgameCount int64 `gorm:"column:galgame_count" json:"galgame_count"`
	LikeCount    int64 `gorm:"column:like_count" json:"like_count"`
}
