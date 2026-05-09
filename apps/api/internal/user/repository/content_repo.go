package repository

import (
	"kun-galgame-api/internal/user/dto"

	"gorm.io/gorm"
)

// UserContentRepository owns the paginated list queries for a user's
// published / liked / favorited / commented content across topics,
// replies, comments, resources, ratings and galgame IDs.
type UserContentRepository struct {
	db *gorm.DB
}

func NewUserContentRepository(db *gorm.DB) *UserContentRepository {
	return &UserContentRepository{db: db}
}

func (r *UserContentRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Galgame IDs
// ──────────────────────────────────────────

func (r *UserContentRepository) FindUserGalgameIDs(uid int, queryType string, page, limit int) ([]int, int64, error) {
	offset := (page - 1) * limit
	var total int64

	baseQuery := r.db.Table("galgame").Select("galgame.id")

	switch queryType {
	case "galgame_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_like ON galgame_like.galgame_id = galgame.id").
			Where("galgame_like.user_id = ?", uid)
	case "galgame_favorite":
		baseQuery = baseQuery.
			Joins("JOIN galgame_favorite ON galgame_favorite.galgame_id = galgame.id").
			Where("galgame_favorite.user_id = ?", uid)
	case "galgame_comment":
		baseQuery = baseQuery.
			Joins("JOIN galgame_comment ON galgame_comment.galgame_id = galgame.id").
			Where("galgame_comment.user_id = ?", uid).
			Group("galgame.id")
	case "galgame_comment_target":
		// Comments targeting this user's galgame comments
		baseQuery = baseQuery.
			Joins("JOIN galgame_comment ON galgame_comment.galgame_id = galgame.id").
			Where("galgame_comment.target_user_id = ? AND galgame_comment.user_id != ?", uid, uid).
			Group("galgame.id")
	case "galgame_comment_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_comment ON galgame_comment.galgame_id = galgame.id").
			Joins("JOIN galgame_comment_like ON galgame_comment_like.galgame_comment_id = galgame_comment.id").
			Where("galgame_comment_like.user_id = ?", uid).
			Group("galgame.id")
	default:
		return []int{}, 0, nil
	}

	baseQuery.Count(&total)

	type idRow struct {
		ID int `gorm:"column:id"`
	}
	var rows []idRow
	err := baseQuery.Order("galgame.created DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, total, nil
}

// ──────────────────────────────────────────
// Topics
// ──────────────────────────────────────────

func (r *UserContentRepository) FindUserTopics(uid int, queryType string, page, limit int) ([]dto.UserTopic, int64, error) {
	offset := (page - 1) * limit
	var results []dto.UserTopic
	var total int64

	baseQuery := r.db.Table("topic").
		Select("topic.id, topic.title, topic.created")

	switch queryType {
	case "topic":
		baseQuery = baseQuery.Where("topic.user_id = ?", uid)
	case "topic_like":
		baseQuery = baseQuery.
			Joins("JOIN topic_like ON topic_like.topic_id = topic.id").
			Where("topic_like.user_id = ?", uid)
	case "topic_upvote":
		baseQuery = baseQuery.
			Joins("JOIN topic_upvote ON topic_upvote.topic_id = topic.id").
			Where("topic_upvote.user_id = ?", uid)
	case "topic_favorite":
		baseQuery = baseQuery.
			Joins("JOIN topic_favorite ON topic_favorite.topic_id = topic.id").
			Where("topic_favorite.user_id = ?", uid)
	case "topic_hide":
		baseQuery = baseQuery.Where("topic.user_id = ? AND topic.status = 1", uid)
	default:
		baseQuery = baseQuery.Where("topic.user_id = ?", uid)
	}

	baseQuery.Count(&total)
	err := baseQuery.Order("topic.created DESC").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}

// ──────────────────────────────────────────
// Replies
// ──────────────────────────────────────────

type UserReply struct {
	TopicID int    `gorm:"column:topic_id" json:"topicId"`
	Content string `gorm:"column:content" json:"content"`
	Created string `gorm:"column:created" json:"created"`
}

func (r *UserContentRepository) FindUserReplies(uid int, queryType string, page, limit int) ([]UserReply, int64, error) {
	offset := (page - 1) * limit
	var results []UserReply
	var total int64

	baseQuery := r.db.Table("topic_reply").
		Select("topic_reply.topic_id, topic_reply.content, topic_reply.created")

	switch queryType {
	case "reply_target":
		baseQuery = baseQuery.
			Where("topic_reply.topic_id IN (SELECT id FROM topic WHERE user_id = ?) AND topic_reply.user_id != ?", uid, uid)
	case "reply_like":
		baseQuery = baseQuery.
			Joins("JOIN topic_reply_like ON topic_reply_like.topic_reply_id = topic_reply.id").
			Where("topic_reply_like.user_id = ?", uid)
	default: // reply_created
		baseQuery = baseQuery.Where("topic_reply.user_id = ?", uid)
	}

	baseQuery.Count(&total)
	err := baseQuery.Order("topic_reply.created DESC").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}

// ──────────────────────────────────────────
// Comments
// ──────────────────────────────────────────

type UserComment struct {
	TopicID int    `gorm:"column:topic_id" json:"topicId"`
	Content string `gorm:"column:content" json:"content"`
	Created string `gorm:"column:created" json:"created"`
}

func (r *UserContentRepository) FindUserComments(uid int, queryType string, page, limit int) ([]UserComment, int64, error) {
	offset := (page - 1) * limit
	var results []UserComment
	var total int64

	baseQuery := r.db.Table("topic_comment").
		Select("topic_comment.topic_id, topic_comment.content, topic_comment.created")

	switch queryType {
	case "comment_target":
		baseQuery = baseQuery.
			Where("topic_comment.target_user_id = ? AND topic_comment.user_id != ?", uid, uid)
	case "comment_like":
		baseQuery = baseQuery.
			Joins("JOIN topic_comment_like ON topic_comment_like.topic_comment_id = topic_comment.id").
			Where("topic_comment_like.user_id = ?", uid)
	default: // comment_created
		baseQuery = baseQuery.Where("topic_comment.user_id = ?", uid)
	}

	baseQuery.Count(&total)
	err := baseQuery.Order("topic_comment.created DESC").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}

// ──────────────────────────────────────────
// Galgame resources
// ──────────────────────────────────────────

type UserResource struct {
	ID        int    `gorm:"column:id" json:"id"`
	GalgameID int    `gorm:"column:galgame_id" json:"galgameId"`
	Type      string `gorm:"column:type" json:"type"`
	Language  string `gorm:"column:language" json:"language"`
	Platform  string `gorm:"column:platform" json:"platform"`
	Size      string `gorm:"column:size" json:"size"`
	Code      string `gorm:"column:code" json:"code"`
	Password  string `gorm:"column:password" json:"password"`
	Note      string `gorm:"column:note" json:"note"`
	Status    int    `gorm:"column:status" json:"status"`
	Created   string `gorm:"column:created" json:"created"`
}

type ResourceLink struct {
	ResourceID int    `gorm:"column:galgame_resource_id"`
	URL        string `gorm:"column:url"`
}

func (r *UserContentRepository) FindUserResources(uid int, queryType string, page, limit int) ([]UserResource, int64, error) {
	offset := (page - 1) * limit
	var results []UserResource
	var total int64

	baseQuery := r.db.Table("galgame_resource").
		Select("galgame_resource.id, galgame_resource.galgame_id, galgame_resource.type, galgame_resource.language, galgame_resource.platform, galgame_resource.size, galgame_resource.code, galgame_resource.password, galgame_resource.note, galgame_resource.status, galgame_resource.created")

	switch queryType {
	case "expire":
		baseQuery = baseQuery.Where("galgame_resource.user_id = ? AND galgame_resource.status = 1", uid)
	case "galgame_resource_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_resource_like ON galgame_resource_like.galgame_resource_id = galgame_resource.id").
			Where("galgame_resource_like.user_id = ?", uid)
	default: // valid
		baseQuery = baseQuery.Where("galgame_resource.user_id = ? AND galgame_resource.status = 0", uid)
	}

	baseQuery.Count(&total)
	err := baseQuery.Order("galgame_resource.created DESC").Offset(offset).Limit(limit).Scan(&results).Error
	return results, total, err
}

func (r *UserContentRepository) FindResourceLinks(resourceIDs []int) (map[int][]string, error) {
	var links []ResourceLink
	err := r.db.Table("galgame_resource_link").
		Select("galgame_resource_id, url").
		Where("galgame_resource_id IN ?", resourceIDs).
		Scan(&links).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]string)
	for _, l := range links {
		result[l.ResourceID] = append(result[l.ResourceID], l.URL)
	}
	return result, nil
}

// ──────────────────────────────────────────
// Galgame ratings
// ──────────────────────────────────────────

// UserRating is one rating row. Identity (UserName/UserAvatar) is hydrated
// at the service layer via userclient.
type UserRating struct {
	ID           int    `gorm:"column:id" json:"id"`
	GalgameID    int    `gorm:"column:galgame_id" json:"galgameId"`
	Recommend    string `gorm:"column:recommend" json:"recommend"`
	Overall      int    `gorm:"column:overall" json:"overall"`
	View         int    `gorm:"column:view" json:"view"`
	Art          int    `gorm:"column:art" json:"art"`
	Story        int    `gorm:"column:story" json:"story"`
	Music        int    `gorm:"column:music" json:"music"`
	Character    int    `gorm:"column:character" json:"character"`
	Route        int    `gorm:"column:route" json:"route"`
	System       int    `gorm:"column:system" json:"system"`
	Voice        int    `gorm:"column:voice" json:"voice"`
	ReplayValue  int    `gorm:"column:replay_value" json:"replay_value"`
	GalgameType  string `gorm:"column:galgame_type" json:"-"` // raw JSON
	PlayStatus   string `gorm:"column:play_status" json:"play_status"`
	SpoilerLevel string `gorm:"column:spoiler_level" json:"spoiler_level"`
	LikeCount    int    `gorm:"column:like_count" json:"likeCount"`
	UserID       int    `gorm:"column:user_id" json:"-"`
	Created      string `gorm:"column:created" json:"created"`
	Updated      string `gorm:"column:updated" json:"updated"`
}

func (r *UserContentRepository) FindUserRatings(uid int, page, limit int) ([]UserRating, int64, error) {
	offset := (page - 1) * limit
	var results []UserRating
	var total int64

	r.db.Table("galgame_rating").Where("user_id = ?", uid).Count(&total)

	err := r.db.Table("galgame_rating").
		Select(`galgame_rating.id, galgame_rating.galgame_id, galgame_rating.recommend, galgame_rating.overall, galgame_rating.view,
			galgame_rating.art, galgame_rating.story, galgame_rating.music, galgame_rating.character, galgame_rating.route, galgame_rating.system, galgame_rating.voice, galgame_rating.replay_value,
			galgame_rating.galgame_type, galgame_rating.play_status, galgame_rating.spoiler_level, galgame_rating.like_count,
			galgame_rating.user_id,
			galgame_rating.created, galgame_rating.updated`).
		Where("galgame_rating.user_id = ?", uid).
		Order("galgame_rating.created DESC").Offset(offset).Limit(limit).
		Scan(&results).Error
	return results, total, err
}

// ──────────────────────────────────────────
// Galgame local stats + resource meta (enrichment for user galgame list)
// ──────────────────────────────────────────

// GalgameLocalStats is a lightweight (view, like_count) row for a galgame.
type GalgameLocalStats struct {
	ID        int `gorm:"column:id"`
	View      int `gorm:"column:view"`
	LikeCount int `gorm:"column:like_count"`
}

// FindGalgameLocalStats batch-loads local (view, like_count) for galgame IDs.
func (r *UserContentRepository) FindGalgameLocalStats(ids []int) map[int]GalgameLocalStats {
	if len(ids) == 0 {
		return map[int]GalgameLocalStats{}
	}
	var rows []GalgameLocalStats
	r.db.Table("galgame").Select("id, view, like_count").
		Where("id IN ?", ids).Scan(&rows)
	out := make(map[int]GalgameLocalStats, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

// GalgameResourceMeta is a (galgame_id, platform, language) tuple distilled
// from galgame_resource rows — used to derive per-galgame platform/language
// sets on the user galgame list.
type GalgameResourceMeta struct {
	GalgameID int    `gorm:"column:galgame_id"`
	Platform  string `gorm:"column:platform"`
	Language  string `gorm:"column:language"`
}

// FindResourceMetaByGalgameIDs loads distinct (platform, language) tuples
// across galgame_resource for the given galgame IDs.
func (r *UserContentRepository) FindResourceMetaByGalgameIDs(ids []int) []GalgameResourceMeta {
	if len(ids) == 0 {
		return nil
	}
	var rows []GalgameResourceMeta
	r.db.Table("galgame_resource").
		Select("DISTINCT galgame_id, platform, language").
		Where("galgame_id IN ?", ids).Scan(&rows)
	return rows
}
