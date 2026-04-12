package common

import (
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	homeGalgameLimit = 12
	homeTopicLimit   = 10
)

type HomeHandler struct {
	db *gorm.DB
}

func NewHomeHandler(db *gorm.DB) *HomeHandler {
	return &HomeHandler{db: db}
}

// ──────────────────────────────────────────
// Response types
// ──────────────────────────────────────────

type HomeLocaleName struct {
	EnUS string `json:"en-us"`
	JaJP string `json:"ja-jp"`
	ZhCN string `json:"zh-cn"`
	ZhTW string `json:"zh-tw"`
}

type HomeBriefUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type HomeGalgame struct {
	ID                 int            `json:"id"`
	Name               HomeLocaleName `json:"name"`
	Banner             string         `json:"banner"`
	User               HomeBriefUser  `json:"user"`
	ContentLimit       string         `json:"contentLimit"`
	View               int            `json:"view"`
	LikeCount          int            `json:"likeCount"`
	ResourceUpdateTime time.Time      `json:"resourceUpdateTime"`
	Platform           []string       `json:"platform"`
	Language           []string       `json:"language"`
}

type HomeTopic struct {
	ID               int           `json:"id"`
	Title            string        `json:"title"`
	View             int           `json:"view"`
	LikeCount        int           `json:"likeCount"`
	ReplyCount       int           `json:"replyCount"`
	CommentCount     int           `json:"commentCount"`
	HasBestAnswer    bool          `json:"hasBestAnswer"`
	IsPollTopic      bool          `json:"isPollTopic"`
	IsNSFWTopic      bool          `json:"isNSFWTopic"`
	Section          []string      `json:"section"`
	Tag              []string      `json:"tag"`
	User             HomeBriefUser `json:"user"`
	Status           int           `json:"status"`
	UpvoteTime       *time.Time    `json:"upvoteTime"`
	StatusUpdateTime time.Time     `json:"statusUpdateTime"`
}

type HomeResponse struct {
	Galgames []HomeGalgame `json:"galgames"`
	Topics   []HomeTopic   `json:"topics"`
}

// ──────────────────────────────────────────
// Internal query result types
// ──────────────────────────────────────────

type galgameRow struct {
	ID                 int       `gorm:"column:id"`
	NameEnUS           string    `gorm:"column:name_en_us"`
	NameJaJP           string    `gorm:"column:name_ja_jp"`
	NameZhCN           string    `gorm:"column:name_zh_cn"`
	NameZhTW           string    `gorm:"column:name_zh_tw"`
	Banner             string    `gorm:"column:banner"`
	ContentLimit       string    `gorm:"column:content_limit"`
	View               int       `gorm:"column:view"`
	LikeCount          int       `gorm:"column:like_count"`
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time"`
	UserID             int       `gorm:"column:user_id"`
	UserName           string    `gorm:"column:user_name"`
	UserAvatar         string    `gorm:"column:user_avatar"`
}

type topicRow struct {
	ID               int        `gorm:"column:id"`
	Title            string     `gorm:"column:title"`
	View             int        `gorm:"column:view"`
	IsNSFW           bool       `gorm:"column:is_nsfw"`
	Status           int        `gorm:"column:status"`
	LikeCount        int        `gorm:"column:like_count"`
	ReplyCount       int        `gorm:"column:reply_count"`
	CommentCount     int        `gorm:"column:comment_count"`
	BestAnswerID     *int       `gorm:"column:best_answer_id"`
	UpvoteTime       *time.Time `gorm:"column:upvote_time"`
	StatusUpdateTime time.Time  `gorm:"column:status_update_time"`
	UserID           int        `gorm:"column:user_id"`
	UserName         string     `gorm:"column:user_name"`
	UserAvatar       string     `gorm:"column:user_avatar"`
}

type resourcePlatformLang struct {
	GalgameID int    `gorm:"column:galgame_id"`
	Platform  string `gorm:"column:platform"`
	Language  string `gorm:"column:language"`
}

// ──────────────────────────────────────────
// Handler
// ──────────────────────────────────────────

// GetHome returns homepage data: galgames + topics.
// GET /api/home
func (h *HomeHandler) GetHome(c *fiber.Ctx) error {
	nsfw := c.Cookies("kun_content_limit", "sfw")
	isSFW := nsfw == "sfw"

	galgames, err := h.getHomeGalgames(isSFW)
	if err != nil {
		return response.Error(c, errors.ErrInternal("获取首页 Galgame 失败"))
	}

	topics, err := h.getHomeTopics(isSFW)
	if err != nil {
		return response.Error(c, errors.ErrInternal("获取首页话题失败"))
	}

	return response.OK(c, HomeResponse{
		Galgames: galgames,
		Topics:   topics,
	})
}

func (h *HomeHandler) getHomeGalgames(isSFW bool) ([]HomeGalgame, error) {
	var rows []galgameRow

	query := h.db.Table("galgame").
		Select(`galgame.id, galgame.name_en_us, galgame.name_ja_jp, galgame.name_zh_cn, galgame.name_zh_tw,
			galgame.banner, galgame.content_limit, galgame.view, galgame.like_count,
			galgame.resource_update_time, galgame.user_id,
			"user".name AS user_name, "user".avatar AS user_avatar`).
		Joins(`JOIN "user" ON "user".id = galgame.user_id`).
		Where("galgame.status != 1").
		Order("galgame.resource_update_time DESC").
		Limit(homeGalgameLimit)

	if isSFW {
		query = query.Where("galgame.content_limit = 'sfw'")
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	// Batch fetch platforms/languages for these galgames
	galgameIDs := make([]int, len(rows))
	for i, r := range rows {
		galgameIDs[i] = r.ID
	}

	var resources []resourcePlatformLang
	if len(galgameIDs) > 0 {
		h.db.Table("galgame_resource").
			Select("galgame_id, platform, language").
			Where("galgame_id IN ?", galgameIDs).
			Find(&resources)
	}

	// Group by galgame_id
	platformMap := map[int]map[string]bool{}
	languageMap := map[int]map[string]bool{}
	for _, r := range resources {
		if platformMap[r.GalgameID] == nil {
			platformMap[r.GalgameID] = map[string]bool{}
		}
		if languageMap[r.GalgameID] == nil {
			languageMap[r.GalgameID] = map[string]bool{}
		}
		platformMap[r.GalgameID][r.Platform] = true
		languageMap[r.GalgameID][r.Language] = true
	}

	result := make([]HomeGalgame, len(rows))
	for i, r := range rows {
		result[i] = HomeGalgame{
			ID: r.ID,
			Name: HomeLocaleName{
				EnUS: r.NameEnUS,
				JaJP: r.NameJaJP,
				ZhCN: r.NameZhCN,
				ZhTW: r.NameZhTW,
			},
			Banner:             r.Banner,
			User:               HomeBriefUser{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			ContentLimit:       r.ContentLimit,
			View:               r.View,
			LikeCount:          r.LikeCount,
			ResourceUpdateTime: r.ResourceUpdateTime,
			Platform:           mapKeys(platformMap[r.ID]),
			Language:           mapKeys(languageMap[r.ID]),
		}
	}

	return result, nil
}

func (h *HomeHandler) getHomeTopics(isSFW bool) ([]HomeTopic, error) {
	var rows []topicRow
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	excludedSections := []string{"g-seeking", "g-other", "t-help"}

	query := h.db.Table("topic").
		Select(`topic.id, topic.title, topic.view, topic.is_nsfw, topic.status,
			topic.like_count, topic.reply_count, topic.comment_count,
			topic.best_answer_id, topic.upvote_time, topic.status_update_time,
			topic.user_id, "user".name AS user_name, "user".avatar AS user_avatar`).
		Joins(`JOIN "user" ON "user".id = topic.user_id`).
		Where("topic.status != 1").
		Where(`topic.id NOT IN (
			SELECT tsr.topic_id FROM topic_section_relation tsr
			JOIN topic_section ts ON ts.id = tsr.topic_section_id
			WHERE ts.name IN ?
		)`, excludedSections).
		Where(`(topic.edited >= ? OR (topic.edited IS NULL AND topic.created >= ?))`, threeMonthsAgo, threeMonthsAgo).
		Order("topic.status_update_time DESC").
		Limit(homeTopicLimit)

	if isSFW {
		query = query.Where("topic.is_nsfw = false")
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	// Fetch sections for these topics
	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	type sectionRow struct {
		TopicID     int    `gorm:"column:topic_id"`
		SectionName string `gorm:"column:name"`
	}
	var sections []sectionRow
	if len(topicIDs) > 0 {
		h.db.Table("topic_section_relation tsr").
			Select("tsr.topic_id, ts.name").
			Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
			Where("tsr.topic_id IN ?", topicIDs).
			Find(&sections)
	}
	sectionMap := map[int][]string{}
	for _, s := range sections {
		sectionMap[s.TopicID] = append(sectionMap[s.TopicID], s.SectionName)
	}

	// Fetch tags for these topics
	type tagRow struct {
		TopicID int    `gorm:"column:topic_id"`
		TagName string `gorm:"column:name"`
	}
	var tags []tagRow
	if len(topicIDs) > 0 {
		h.db.Table("topic_tag_relation ttr").
			Select("ttr.topic_id, tt.name").
			Joins("JOIN topic_tag tt ON tt.id = ttr.tag_id").
			Where("ttr.topic_id IN ?", topicIDs).
			Find(&tags)
	}
	tagMap := map[int][]string{}
	for _, t := range tags {
		tagMap[t.TopicID] = append(tagMap[t.TopicID], t.TagName)
	}

	result := make([]HomeTopic, len(rows))
	for i, r := range rows {
		topicTags := tagMap[r.ID]
		if topicTags == nil {
			topicTags = []string{}
		}
		topicSections := sectionMap[r.ID]
		if topicSections == nil {
			topicSections = []string{}
		}

		result[i] = HomeTopic{
			ID:               r.ID,
			Title:            r.Title,
			View:             r.View,
			LikeCount:        r.LikeCount,
			ReplyCount:       r.ReplyCount,
			CommentCount:     r.CommentCount,
			HasBestAnswer:    r.BestAnswerID != nil,
			IsPollTopic:      false,
			IsNSFWTopic:      r.IsNSFW,
			Section:          topicSections,
			Tag:              topicTags,
			User:             HomeBriefUser{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Status:           r.Status,
			UpvoteTime:       r.UpvoteTime,
			StatusUpdateTime: r.StatusUpdateTime,
		}
	}

	return result, nil
}

func mapKeys(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
