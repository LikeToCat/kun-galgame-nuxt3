package model

import (
	"time"

	"encoding/json"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────
// Core
// ──────────────────────────────────────────

type Galgame struct {
	ID                 int       `gorm:"primaryKey;autoIncrement" json:"id"`
	VndbID             string    `gorm:"column:vndb_id;uniqueIndex;type:varchar(10)" json:"vndb_id"`
	NameEnUS           string    `gorm:"column:name_en_us;type:varchar(1000);default:''" json:"name_en_us"`
	NameJaJP           string    `gorm:"column:name_ja_jp;type:varchar(1000);default:''" json:"name_ja_jp"`
	NameZhCN           string    `gorm:"column:name_zh_cn;type:varchar(1000);default:''" json:"name_zh_cn"`
	NameZhTW           string    `gorm:"column:name_zh_tw;type:varchar(1000);default:''" json:"name_zh_tw"`
	Banner             string    `gorm:"type:varchar(233);default:''" json:"banner"`
	IntroEnUS          string    `gorm:"column:intro_en_us;type:text;default:''" json:"intro_en_us"`
	IntroJaJP          string    `gorm:"column:intro_ja_jp;type:text;default:''" json:"intro_ja_jp"`
	IntroZhCN          string    `gorm:"column:intro_zh_cn;type:text;default:''" json:"intro_zh_cn"`
	IntroZhTW          string    `gorm:"column:intro_zh_tw;type:text;default:''" json:"intro_zh_tw"`
	ContentLimit       string    `gorm:"column:content_limit;type:varchar(10);default:'sfw'" json:"content_limit"`
	Status             int       `gorm:"default:0" json:"status"`
	View               int       `gorm:"default:0" json:"view"`
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time;autoCreateTime" json:"resource_update_time"`
	OriginalLanguage   string    `gorm:"column:original_language;default:'ja-jp'" json:"original_language"`
	AgeLimit           string    `gorm:"column:age_limit;default:'r18'" json:"age_limit"`

	UserID   int  `gorm:"column:user_id;not null" json:"user_id"`
	SeriesID *int `gorm:"column:series_id" json:"series_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`

	// Counts (denormalized, maintained via transactions)
	LikeCount        int `gorm:"column:like_count;default:0" json:"like_count"`
	FavoriteCount    int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	ResourceCount    int `gorm:"column:resource_count;default:0" json:"resource_count"`
	CommentCount     int `gorm:"column:comment_count;default:0" json:"comment_count"`
	ContributorCount int `gorm:"column:contributor_count;default:0" json:"contributor_count"`
	RatingCount      int `gorm:"column:rating_count;default:0" json:"rating_count"`
}

func (Galgame) TableName() string { return "galgame" }

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

type GalgameSeries struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"uniqueIndex;type:varchar(1000);default:''" json:"name"`
	Description string `gorm:"type:varchar(2000);default:''" json:"description"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameSeries) TableName() string { return "galgame_series" }

// ──────────────────────────────────────────
// Alias
// ──────────────────────────────────────────

type GalgameAlias struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"default:''" json:"name"`
	GalgameID int    `gorm:"column:galgame_id;not null" json:"galgame_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameAlias) TableName() string { return "galgame_alias" }

// ──────────────────────────────────────────
// Interactions (like / favorite / contributor)
// ──────────────────────────────────────────

type GalgameLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_like" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameLike) TableName() string { return "galgame_like" }

type GalgameFavorite struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_favorite" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameFavorite) TableName() string { return "galgame_favorite" }

type GalgameContributor struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_contributor" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_contributor" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameContributor) TableName() string { return "galgame_contributor" }

// ──────────────────────────────────────────
// Tag / Official / Engine (metadata entities)
// ──────────────────────────────────────────

type GalgameTag struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Description string `gorm:"default:''" json:"description"`
	Category    string `gorm:"not null" json:"category"` // content, sexual, technical

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameTag) TableName() string { return "galgame_tag" }

type GalgameTagAlias struct {
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"default:''" json:"name"`
	GalgameTagID int    `gorm:"column:galgame_tag_id;not null;uniqueIndex:idx_tag_alias" json:"galgame_tag_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameTagAlias) TableName() string { return "galgame_tag_alias" }

type GalgameOfficial struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Link        string `gorm:"default:''" json:"link"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Category    string `gorm:"not null" json:"category"` // company, individual, amateur
	Lang        string `gorm:"default:''" json:"lang"`
	Description string `gorm:"default:''" json:"description"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameOfficial) TableName() string { return "galgame_official" }

type GalgameOfficialAlias struct {
	ID                 int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string `gorm:"default:''" json:"name"`
	GalgameOfficialID  int    `gorm:"column:galgame_official_id;not null;uniqueIndex:idx_official_alias" json:"galgame_official_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameOfficialAlias) TableName() string { return "galgame_official_alias" }

type GalgameEngine struct {
	ID          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `gorm:"default:''" json:"description"`
	Alias       json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"alias"` // String[] → jsonb

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameEngine) TableName() string { return "galgame_engine" }

// ──────────────────────────────────────────
// Relation tables (many-to-many with composite PK)
// ──────────────────────────────────────────

type GalgameTagRelation struct {
	GalgameID    int `gorm:"column:galgame_id;primaryKey" json:"galgame_id"`
	TagID        int `gorm:"column:tag_id;primaryKey" json:"tag_id"`
	SpoilerLevel int `gorm:"column:spoiler_level;default:0" json:"spoiler_level"` // 0=none, 1=mild, 2=severe

	Tag GalgameTag `gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE" json:"tag,omitzero"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameTagRelation) TableName() string { return "galgame_tag_relation" }

type GalgameOfficialRelation struct {
	GalgameID  int `gorm:"column:galgame_id;primaryKey" json:"galgame_id"`
	OfficialID int `gorm:"column:official_id;primaryKey" json:"official_id"`

	Official GalgameOfficial `gorm:"foreignKey:OfficialID;constraint:OnDelete:CASCADE" json:"official,omitzero"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameOfficialRelation) TableName() string { return "galgame_official_relation" }

type GalgameEngineRelation struct {
	GalgameID int `gorm:"column:galgame_id;primaryKey" json:"galgame_id"`
	EngineID  int `gorm:"column:engine_id;primaryKey" json:"engine_id"`

	Engine GalgameEngine `gorm:"foreignKey:EngineID;constraint:OnDelete:CASCADE" json:"engine,omitzero"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameEngineRelation) TableName() string { return "galgame_engine_relation" }

// ──────────────────────────────────────────
// Link
// ──────────────────────────────────────────

type GalgameLink struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"type:varchar(107);default:''" json:"name"`
	Link      string `gorm:"type:varchar(233);default:''" json:"link"`
	GalgameID int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID    int    `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameLink) TableName() string { return "galgame_link" }

// ──────────────────────────────────────────
// PR (Pull Request) & History
// ──────────────────────────────────────────

type GalgamePR struct {
	ID            int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Status        int             `gorm:"default:0" json:"status"` // 0=pending, 1=merged, 2=declined
	Index         int             `gorm:"default:0" json:"index"`
	Note          string          `gorm:"type:varchar(1007);default:''" json:"note"`
	CompletedTime *time.Time      `gorm:"column:completed_time" json:"completed_time"`
	OldData       json.RawMessage `gorm:"column:old_data;type:jsonb" json:"old_data"`
	NewData       json.RawMessage `gorm:"column:new_data;type:jsonb" json:"new_data"`
	UserID        int             `gorm:"column:user_id;not null" json:"user_id"`
	GalgameID     int             `gorm:"column:galgame_id;not null" json:"galgame_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgamePR) TableName() string { return "galgame_pr" }

type GalgameHistory struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Action    string `gorm:"default:''" json:"action"`
	Type      string `gorm:"default:''" json:"type"`
	Content   string `gorm:"type:varchar(1007);default:''" json:"content"`
	GalgameID int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID    int    `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameHistory) TableName() string { return "galgame_history" }

// ──────────────────────────────────────────
// Resource
// ──────────────────────────────────────────

type GalgameResource struct {
	ID         int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Type       string     `gorm:"default:''" json:"type"`
	Language   string     `gorm:"default:''" json:"language"`
	Platform   string     `gorm:"default:''" json:"platform"`
	Size       string     `gorm:"type:varchar(107);default:''" json:"size"`
	Code       string     `gorm:"type:varchar(1007);default:''" json:"code"`
	Password   string     `gorm:"type:varchar(1007);default:''" json:"password"`
	Note       string     `gorm:"type:varchar(1007);default:''" json:"note"`
	UpdateTime time.Time  `gorm:"column:update_time;autoCreateTime" json:"update_time"`
	View       int        `gorm:"default:0" json:"view"`
	Status     int        `gorm:"default:0" json:"status"`
	Download   int        `gorm:"default:0" json:"download"`
	Edited     *time.Time `gorm:"" json:"edited"`
	GalgameID  int        `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID     int        `gorm:"column:user_id;not null" json:"user_id"`

	LikeCount int `gorm:"column:like_count;default:0" json:"like_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResource) TableName() string { return "galgame_resource" }

// GalgameResourceProvider replaces the old provider String[] field.
// Each row represents one provider for a resource.
type GalgameResourceProvider struct {
	ID         int    `gorm:"primaryKey;autoIncrement" json:"id"`
	ResourceID int    `gorm:"column:resource_id;not null;uniqueIndex:idx_resource_provider" json:"resource_id"`
	Name       string `gorm:"not null;uniqueIndex:idx_resource_provider" json:"name"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceProvider) TableName() string { return "galgame_resource_provider" }

type GalgameResourceLink struct {
	ID         int    `gorm:"primaryKey;autoIncrement" json:"id"`
	URL        string `gorm:"column:url;not null" json:"url"`
	ResourceID int    `gorm:"column:galgame_resource_id;not null;uniqueIndex:idx_resource_link" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLink) TableName() string { return "galgame_resource_link" }

type GalgameResourceLike struct {
	ID         int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int `gorm:"column:user_id;not null;uniqueIndex:idx_resource_like" json:"user_id"`
	ResourceID int `gorm:"column:galgame_resource_id;not null;uniqueIndex:idx_resource_like" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLike) TableName() string { return "galgame_resource_like" }

// ──────────────────────────────────────────
// Comment
// ──────────────────────────────────────────

type GalgameComment struct {
	ID           int  `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string `gorm:"type:varchar(1007);not null" json:"content"`
	GalgameID    int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID       int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID *int   `gorm:"column:target_user_id" json:"target_user_id"`

	LikeCount int `gorm:"column:like_count;default:0" json:"like_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameComment) TableName() string { return "galgame_comment" }

type GalgameCommentLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_comment_like" json:"user_id"`
	CommentID int `gorm:"column:galgame_comment_id;not null;uniqueIndex:idx_comment_like" json:"galgame_comment_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameCommentLike) TableName() string { return "galgame_comment_like" }

// ──────────────────────────────────────────
// Rating
// ──────────────────────────────────────────

type GalgameRating struct {
	ID            int            `gorm:"primaryKey;autoIncrement" json:"id"`
	Recommend     string         `gorm:"not null" json:"recommend"`           // strong_no, no, neutral, yes, strong_yes
	Overall       int            `gorm:"not null" json:"overall"`             // 1-10
	View          int            `gorm:"default:0" json:"view"`
	GalgameType   json.RawMessage `gorm:"column:galgame_type;type:jsonb;default:'[]'" json:"galgame_type"` // String[] → jsonb
	PlayStatus    string         `gorm:"column:play_status;default:'not_started'" json:"play_status"`
	ShortSummary  string         `gorm:"column:short_summary;type:varchar(1314);default:''" json:"short_summary"`
	SpoilerLevel  string         `gorm:"column:spoiler_level;default:'none'" json:"spoiler_level"`

	// Dimension scores (1-10, 0 = not rated)
	Art          int `gorm:"default:0" json:"art"`
	Story        int `gorm:"default:0" json:"story"`
	Music        int `gorm:"default:0" json:"music"`
	Character    int `gorm:"default:0" json:"character"`
	Route        int `gorm:"default:0" json:"route"`
	System       int `gorm:"default:0" json:"system"`
	Voice        int `gorm:"default:0" json:"voice"`
	ReplayValue  int `gorm:"column:replay_value;default:0" json:"replay_value"`

	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_rating_user_galgame" json:"user_id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_rating_user_galgame;index" json:"galgame_id"`

	LikeCount    int `gorm:"column:like_count;default:0" json:"like_count"`
	CommentCount int `gorm:"column:comment_count;default:0" json:"comment_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRating) TableName() string { return "galgame_rating" }

type GalgameRatingLike struct {
	ID       int `gorm:"primaryKey;autoIncrement" json:"id"`
	RatingID int `gorm:"column:galgame_rating_id;not null;uniqueIndex:idx_rating_like" json:"galgame_rating_id"`
	UserID   int `gorm:"column:user_id;not null;uniqueIndex:idx_rating_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRatingLike) TableName() string { return "galgame_rating_like" }

type GalgameRatingComment struct {
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string `gorm:"type:varchar(1314);default:''" json:"content"`
	RatingID     int    `gorm:"column:galgame_rating_id;not null" json:"galgame_rating_id"`
	UserID       int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID *int   `gorm:"column:target_user_id" json:"target_user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameRatingComment) TableName() string { return "galgame_rating_comment" }

// ──────────────────────────────────────────
// GORM callbacks: auto-maintain updated_at
// ──────────────────────────────────────────

func autoUpdateTimestamp(db *gorm.DB) {
	db.Statement.SetColumn("updated", time.Now())
}
