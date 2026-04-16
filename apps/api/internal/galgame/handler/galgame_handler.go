package handler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/middleware"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type GalgameHandler struct {
	db            *gorm.DB
	galgameClient *client.GalgameClient
}

func NewGalgameHandler(db *gorm.DB, gc *client.GalgameClient) *GalgameHandler {
	return &GalgameHandler{db: db, galgameClient: gc}
}

// getAccessToken extracts OAuth access_token from the session.
func getAccessToken(c *fiber.Ctx) string {
	// The access token is stored in the session data in Redis.
	// For wiki service calls, we need the OAuth access_token.
	// Currently the middleware stores it in session; we need to expose it.
	// For now, check if there's a header or cookie we can use.
	// The session's OAuthAccessToken is in Redis — retrieve from middleware.
	return c.Get("X-OAuth-Token") // frontend must send this header for wiki proxy calls
}

// ──────────────────────────────────────────
// Proxy endpoints (forward to wiki service with local side effects)
// ──────────────────────────────────────────

// Create proxies galgame creation to wiki service, then adds local side effects.
// POST /api/galgame
func (h *GalgameHandler) Create(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := getAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrBadRequest("缺少 OAuth 访问令牌"))
	}

	// Forward to wiki service
	data, appErr := h.galgameClient.PostWithToken(c.Context(), "/galgame", token, json.RawMessage(c.Body()))
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Parse created galgame ID for local side effects
	var created struct {
		ID int `json:"id"`
	}
	json.Unmarshal(data, &created)

	if created.ID > 0 {
		h.db.Transaction(func(tx *gorm.DB) error {
			tx.Create(&model.GalgameLocal{ID: created.ID})
			tx.Model(&userModel.User{}).Where("id = ?", user.UID).
				Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardCreateGalgame))
			return nil
		})
	}

	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// MergePR proxies PR merge to wiki service, then awards moemoepoint.
// PUT /api/galgame/:gid/prs/:id/merge
func (h *GalgameHandler) MergePR(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := getAccessToken(c)
	gid := c.Params("gid")
	prID := c.Params("id")

	// Get PR details first to know who submitted it
	prData, appErr := h.galgameClient.Get(c.Context(), fmt.Sprintf("/galgame/%s/prs/%s", gid, prID), nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var prInfo struct {
		PR struct {
			UserID int `json:"user_id"`
		} `json:"pr"`
	}
	json.Unmarshal(prData, &prInfo)

	// Forward merge to wiki service
	data, appErr := h.galgameClient.PutWithToken(c.Context(), fmt.Sprintf("/galgame/%s/prs/%s/merge", gid, prID), token, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Award moemoepoint to PR submitter
	if prInfo.PR.UserID > 0 && prInfo.PR.UserID != user.UID {
		h.db.Model(&userModel.User{}).Where("id = ?", prInfo.PR.UserID).
			Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardPRMerge))

		gidInt, _ := strconv.Atoi(gid)
		createDedupMessage(h.db, user.UID, prInfo.PR.UserID, "merged", gidInt)
	}

	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// ──────────────────────────────────────────
// Aggregation endpoint (wiki metadata + local stats + interaction)
// ──────────────────────────────────────────

// GetDetail returns galgame metadata from wiki + local stats + user interaction.
// GET /api/galgame/:gid
func (h *GalgameHandler) GetDetail(c *fiber.Ctx) error {
	gid := c.Params("gid")
	userInfo := middleware.GetUser(c)
	gidInt, _ := strconv.Atoi(gid)

	// Fetch wiki metadata
	wikiData, appErr := h.galgameClient.Get(c.Context(), "/galgame/"+gid, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Parse wiki response
	type wikiAlias struct {
		Name string `json:"name"`
	}
	type wikiOfficialAlias struct {
		Name string `json:"name"`
	}
	type wikiOfficial struct {
		ID       int                 `json:"id"`
		Name     string              `json:"name"`
		Link     string              `json:"link"`
		Category string              `json:"category"`
		Lang     string              `json:"lang"`
		Alias    []wikiOfficialAlias `json:"alias"`
	}
	type wikiOfficialRel struct {
		Official wikiOfficial `json:"official"`
	}
	type wikiEngine struct {
		ID    int      `json:"id"`
		Name  string   `json:"name"`
		Alias []string `json:"alias"`
	}
	type wikiEngineRel struct {
		Engine wikiEngine `json:"engine"`
	}
	type wikiTag struct {
		SpoilerLevel int `json:"spoiler_level"`
		Tag          struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"tag"`
	}
	type wikiContributor struct {
		UserID int `json:"user_id"`
	}
	type wikiUser struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type wikiGalgame struct {
		ID                 int                `json:"id"`
		VndbID             string             `json:"vndb_id"`
		NameEnUs           string             `json:"name_en_us"`
		NameJaJp           string             `json:"name_ja_jp"`
		NameZhCn           string             `json:"name_zh_cn"`
		NameZhTw           string             `json:"name_zh_tw"`
		Banner             string             `json:"banner"`
		IntroEnUs          string             `json:"intro_en_us"`
		IntroJaJp          string             `json:"intro_ja_jp"`
		IntroZhCn          string             `json:"intro_zh_cn"`
		IntroZhTw          string             `json:"intro_zh_tw"`
		ContentLimit       string             `json:"content_limit"`
		View               int                `json:"view"`
		ResourceUpdateTime string             `json:"resource_update_time"`
		OriginalLanguage   string             `json:"original_language"`
		AgeLimit           string             `json:"age_limit"`
		UserID             int                `json:"user_id"`
		SeriesID           *int               `json:"series_id"`
		Status             int                `json:"status"`
		Alias              []wikiAlias        `json:"alias"`
		Official           []wikiOfficialRel  `json:"official"`
		Engine             []wikiEngineRel    `json:"engine"`
		Tag                []wikiTag          `json:"tag"`
		Contributor        []wikiContributor  `json:"contributor"`
		Created            string             `json:"created"`
		Updated            string             `json:"updated"`
	}
	var parsed struct {
		Galgame wikiGalgame         `json:"galgame"`
		Users   map[string]wikiUser `json:"users"`
	}
	if err := json.Unmarshal(wikiData, &parsed); err != nil {
		return response.Error(c, errors.ErrInternal("解析 Wiki 响应失败"))
	}

	g := parsed.Galgame
	if g.Status == 1 {
		return response.Error(c, errors.ErrNotFound("该 Galgame 已被封禁"))
	}

	// Increment view
	go h.db.Table("galgame").Where("id = ?", gidInt).
		Update("view", gorm.Expr("view + 1"))

	// Local stats
	var local model.GalgameLocal
	h.db.Where("id = ?", gidInt).First(&local)

	// User interaction
	isLiked, isFavorited := false, false
	if userInfo != nil {
		var lc, fc int64
		h.db.Model(&model.GalgameLike{}).Where("user_id = ? AND galgame_id = ?", userInfo.UID, gidInt).Count(&lc)
		h.db.Model(&model.GalgameFavorite{}).Where("user_id = ? AND galgame_id = ?", userInfo.UID, gidInt).Count(&fc)
		isLiked, isFavorited = lc > 0, fc > 0
	}

	// Resolve user from wiki users map
	resolveUser := func(uid int) fiber.Map {
		key := fmt.Sprintf("%d", uid)
		if u, ok := parsed.Users[key]; ok {
			return fiber.Map{"id": u.ID, "name": u.Name, "avatar": u.Avatar}
		}
		return fiber.Map{"id": uid, "name": "", "avatar": ""}
	}

	// Convert introduction markdown to HTML
	introHTML := fiber.Map{
		"en-us": markdown.Render(g.IntroEnUs),
		"ja-jp": markdown.Render(g.IntroJaJp),
		"zh-cn": markdown.Render(g.IntroZhCn),
		"zh-tw": markdown.Render(g.IntroZhTw),
	}

	// Alias → string[]
	aliases := make([]string, len(g.Alias))
	for i, a := range g.Alias {
		aliases[i] = a.Name
	}

	// Contributors
	contributors := make([]fiber.Map, len(g.Contributor))
	for i, c := range g.Contributor {
		contributors[i] = resolveUser(c.UserID)
	}

	// Officials
	officials := make([]fiber.Map, len(g.Official))
	for i, o := range g.Official {
		aliasNames := make([]string, len(o.Official.Alias))
		for j, a := range o.Official.Alias {
			aliasNames[j] = a.Name
		}
		officials[i] = fiber.Map{
			"id": o.Official.ID, "name": o.Official.Name,
			"link": o.Official.Link, "category": o.Official.Category,
			"lang": o.Official.Lang, "alias": aliasNames,
			"galgameCount": 0,
		}
	}

	// Engines
	engines := make([]fiber.Map, len(g.Engine))
	for i, e := range g.Engine {
		engines[i] = fiber.Map{
			"id": e.Engine.ID, "name": e.Engine.Name,
			"alias": e.Engine.Alias, "galgameCount": 0,
		}
	}

	// Tags
	tags := make([]fiber.Map, len(g.Tag))
	for i, t := range g.Tag {
		tags[i] = fiber.Map{
			"id": t.Tag.ID, "name": t.Tag.Name,
			"category": t.Tag.Category, "galgameCount": 0,
			"spoilerLevel": t.SpoilerLevel,
		}
	}

	// Series
	var series any
	if g.SeriesID != nil {
		seriesData, seriesErr := h.galgameClient.Get(
			c.Context(),
			fmt.Sprintf("/series/%d", *g.SeriesID),
			nil,
		)
		if seriesErr == nil {
			type seriesGalgame struct {
				NameEnUs     string `json:"name_en_us"`
				NameJaJp     string `json:"name_ja_jp"`
				NameZhCn     string `json:"name_zh_cn"`
				NameZhTw     string `json:"name_zh_tw"`
				Banner       string `json:"banner"`
				ContentLimit string `json:"content_limit"`
			}
			var s struct {
				ID          int             `json:"id"`
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Galgame     []seriesGalgame `json:"galgame"`
				Created     string          `json:"created"`
				Updated     string          `json:"updated"`
			}
			if err := json.Unmarshal(seriesData, &s); err == nil {
				isNSFW := false
				samples := make([]fiber.Map, 0, min(len(s.Galgame), 5))
				for j, sg := range s.Galgame {
					if sg.ContentLimit == "nsfw" {
						isNSFW = true
					}
					if j < 5 {
						samples = append(samples, fiber.Map{
							"name": fiber.Map{
								"en-us": sg.NameEnUs, "ja-jp": sg.NameJaJp,
								"zh-cn": sg.NameZhCn, "zh-tw": sg.NameZhTw,
							},
							"banner": sg.Banner,
						})
					}
				}
				series = fiber.Map{
					"id": s.ID, "name": s.Name,
					"description":   s.Description,
					"isNSFW":        isNSFW,
					"sampleGalgame": samples,
					"galgameCount":  len(s.Galgame),
					"created":       s.Created,
					"updated":       s.Updated,
				}
			}
		}
	}

	// Platform/Language/Type from local galgame_resource
	type resRow struct {
		Platform string `gorm:"column:platform"`
		Language string `gorm:"column:language"`
		Type     string `gorm:"column:type"`
	}
	var resources []resRow
	h.db.Table("galgame_resource").
		Select("DISTINCT platform, language, type").
		Where("galgame_id = ?", gidInt).Scan(&resources)

	platformSet := map[string]bool{}
	languageSet := map[string]bool{}
	typeSet := map[string]bool{}
	for _, r := range resources {
		if r.Platform != "" {
			platformSet[r.Platform] = true
		}
		if r.Language != "" {
			languageSet[r.Language] = true
		}
		if r.Type != "" {
			typeSet[r.Type] = true
		}
	}

	// Ratings from local DB
	type ratingRow struct {
		ID           int    `gorm:"column:id"`
		Recommend    string `gorm:"column:recommend"`
		Overall      int    `gorm:"column:overall"`
		View         int    `gorm:"column:view"`
		GalgameType  string `gorm:"column:galgame_type"`
		PlayStatus   string `gorm:"column:play_status"`
		ShortSummary string `gorm:"column:short_summary"`
		SpoilerLevel string `gorm:"column:spoiler_level"`
		Art, Story, Music, Character       int
		Route, System, Voice, ReplayValue  int
		LikeCount    int    `gorm:"column:like_count"`
		Created      string `gorm:"column:created"`
		Updated      string `gorm:"column:updated"`
		UserID       int    `gorm:"column:user_id"`
	}
	var ratingRows []ratingRow
	h.db.Table("galgame_rating").
		Where("galgame_id = ?", gidInt).
		Order("created DESC").
		Scan(&ratingRows)

	// Batch fetch rating users
	ratingUserIDs := make([]int, len(ratingRows))
	for i, r := range ratingRows {
		ratingUserIDs[i] = r.UserID
	}
	var ratingUsers []userModel.UserBrief
	if len(ratingUserIDs) > 0 {
		h.db.Where("id IN ?", ratingUserIDs).Find(&ratingUsers)
	}
	ratingUserMap := make(map[int]userModel.UserBrief, len(ratingUsers))
	for _, u := range ratingUsers {
		ratingUserMap[u.ID] = u
	}

	// Check which ratings the current user liked
	ratingIDs := make([]int, len(ratingRows))
	for i, r := range ratingRows {
		ratingIDs[i] = r.ID
	}
	likedRatingSet := map[int]bool{}
	if userInfo != nil && len(ratingIDs) > 0 {
		type likeRow struct {
			GalgameRatingID int `gorm:"column:galgame_rating_id"`
		}
		var likes []likeRow
		h.db.Table("galgame_rating_like").
			Select("galgame_rating_id").
			Where("user_id = ? AND galgame_rating_id IN ?", userInfo.UID, ratingIDs).
			Scan(&likes)
		for _, l := range likes {
			likedRatingSet[l.GalgameRatingID] = true
		}
	}

	ratings := make([]fiber.Map, len(ratingRows))
	for i, r := range ratingRows {
		ratings[i] = fiber.Map{
			"id": r.ID, "user": ratingUserMap[r.UserID],
			"recommend": r.Recommend, "overall": r.Overall, "view": r.View,
			"galgameType": json.RawMessage(r.GalgameType),
			"play_status": r.PlayStatus, "short_summary": r.ShortSummary,
			"spoiler_level": r.SpoilerLevel,
			"art": r.Art, "story": r.Story, "music": r.Music,
			"character": r.Character, "route": r.Route, "system": r.System,
			"voice": r.Voice, "replay_value": r.ReplayValue,
			"likeCount": r.LikeCount, "isLiked": likedRatingSet[r.ID],
			"galgameId": gidInt,
			"created": r.Created, "updated": r.Updated,
			"galgame": fiber.Map{
				"id": g.ID, "contentLimit": g.ContentLimit,
				"name": fiber.Map{
					"en-us": g.NameEnUs, "ja-jp": g.NameJaJp,
					"zh-cn": g.NameZhCn, "zh-tw": g.NameZhTw,
				},
			},
		}
	}

	return response.OK(c, fiber.Map{
		"id":     g.ID,
		"vndbId": g.VndbID,
		"user":   resolveUser(g.UserID),
		"name": fiber.Map{
			"en-us": g.NameEnUs, "ja-jp": g.NameJaJp,
			"zh-cn": g.NameZhCn, "zh-tw": g.NameZhTw,
		},
		"banner":       g.Banner,
		"introduction": introHTML,
		"markdown": fiber.Map{
			"en-us": g.IntroEnUs, "ja-jp": g.IntroJaJp,
			"zh-cn": g.IntroZhCn, "zh-tw": g.IntroZhTw,
		},
		"contentLimit":       g.ContentLimit,
		"resourceUpdateTime": g.ResourceUpdateTime,
		"view":               local.View,
		"originalLanguage":   g.OriginalLanguage,
		"ageLimit":           g.AgeLimit,
		"platform":           mapKeysStr(platformSet),
		"language":           mapKeysStr(languageSet),
		"type":               mapKeysStr(typeSet),
		"contributor":        contributors,
		"likeCount":          local.LikeCount,
		"isLiked":            isLiked,
		"favoriteCount":      local.FavoriteCount,
		"isFavorited":        isFavorited,
		"alias":              aliases,
		"series":             series,
		"engine":             engines,
		"official":           officials,
		"tag":                tags,
		"ratings":            ratings,
		"created":            g.Created,
		"updated":            g.Updated,
	})
}

func mapKeysStr(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ──────────────────────────────────────────
// Local interactions (no wiki service call)
// ──────────────────────────────────────────

// ToggleLike toggles galgame like. Moemoepoint goes to content OWNER.
// PUT /api/galgame/:gid/like
func (h *GalgameHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, _ := strconv.Atoi(c.Params("gid"))

	// Get galgame owner from wiki
	ownerID := h.getGalgameOwner(c, gid)

	if ownerID == user.UID {
		return response.Error(c, errors.ErrBadRequest("您不能给自己点赞"))
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameLike
		result := tx.Where("user_id = ? AND galgame_id = ?", user.UID, gid).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameLike{UserID: user.UID, GalgameID: gid})
			tx.Model(&model.GalgameLocal{}).Where("id = ?", gid).
				Update("like_count", gorm.Expr("like_count + 1"))
			if ownerID > 0 {
				tx.Model(&userModel.User{}).Where("id = ?", ownerID).
					Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
				createDedupMessage(tx, user.UID, ownerID, "liked", gid)
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameLocal{}).Where("id = ?", gid).
				Update("like_count", gorm.Expr("like_count - 1"))
			if ownerID > 0 {
				tx.Model(&userModel.User{}).Where("id = ?", ownerID).
					Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
			}
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite toggles galgame favorite.
// PUT /api/galgame/:gid/favorite
func (h *GalgameHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, _ := strconv.Atoi(c.Params("gid"))

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameFavorite
		result := tx.Where("user_id = ? AND galgame_id = ?", user.UID, gid).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameFavorite{UserID: user.UID, GalgameID: gid})
			tx.Model(&model.GalgameLocal{}).Where("id = ?", gid).
				Update("favorite_count", gorm.Expr("favorite_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameLocal{}).Where("id = ?", gid).
				Update("favorite_count", gorm.Expr("favorite_count - 1"))
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// GetComments returns galgame comments.
// GET /api/galgame/:gid/comment
func (h *GalgameHandler) GetComments(c *fiber.Ctx) error {
	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type dbRow struct {
		ID               int
		Content          string
		GalgameID        int
		UserID           int
		UserName         string
		UserAvatar       string
		TargetUserID     *int
		TargetUserName   string
		TargetUserAvatar string
		LikeCount        int
		CreatedAt        string
	}

	var rows []dbRow
	var total int64

	h.db.Model(&model.GalgameComment{}).Where("galgame_id = ?", gid).Count(&total)
	h.db.Table("galgame_comment gc").
		Select(`gc.id, gc.content, gc.galgame_id, gc.user_id,
			u1.name AS user_name, u1.avatar AS user_avatar,
			gc.target_user_id, u2.name AS target_user_name, u2.avatar AS target_user_avatar,
			gc.like_count, gc.created AS created_at`).
		Joins(`LEFT JOIN "user" u1 ON u1.id = gc.user_id`).
		Joins(`LEFT JOIN "user" u2 ON u2.id = gc.target_user_id`).
		Where("gc.galgame_id = ?", gid).
		Order("gc.created DESC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&rows)

	type userObj struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type commentItem struct {
		ID         int      `json:"id"`
		Content    string   `json:"content"`
		GalgameID  int      `json:"galgameId"`
		User       userObj  `json:"user"`
		TargetUser *userObj `json:"targetUser"`
		LikeCount  int      `json:"likeCount"`
		Created    string   `json:"created"`
	}

	items := make([]commentItem, len(rows))
	for i, r := range rows {
		item := commentItem{
			ID: r.ID, Content: r.Content, GalgameID: r.GalgameID,
			User:      userObj{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			LikeCount: r.LikeCount, Created: r.CreatedAt,
		}
		if r.TargetUserID != nil {
			item.TargetUser = &userObj{ID: *r.TargetUserID, Name: r.TargetUserName, Avatar: r.TargetUserAvatar}
		}
		items[i] = item
	}

	return response.Paginated(c, items, total)
}

// CreateComment creates a galgame comment.
// POST /api/galgame/:gid/comment
func (h *GalgameHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Content      string `json:"content" validate:"required,min=1,max=1007"`
		TargetUserID *int   `json:"target_user_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	comment := model.GalgameComment{
		Content:      req.Content,
		GalgameID:    gid,
		UserID:       user.UID,
		TargetUserID: req.TargetUserID,
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&comment)
		tx.Model(&model.GalgameLocal{}).Where("id = ?", gid).
			Update("comment_count", gorm.Expr("comment_count + 1"))

		if req.TargetUserID != nil && *req.TargetUserID != user.UID {
			tx.Model(&userModel.User{}).Where("id = ?", *req.TargetUserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + 1"))

			link := fmt.Sprintf("/galgame/%d", gid)
			tx.Create(&msgModel.Message{
				SenderID: user.UID, ReceiverID: *req.TargetUserID,
				Type: "commented", Content: truncate(req.Content, 233),
				Link: link, Status: "unread",
			})
		}
		return nil
	})

	// Fetch creator info for response
	var creatorName, creatorAvatar string
	h.db.Table(`"user"`).Select("name, avatar").Where("id = ?", user.UID).Row().Scan(&creatorName, &creatorAvatar)

	type userObj struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	type commentResp struct {
		ID         int      `json:"id"`
		Content    string   `json:"content"`
		GalgameID  int      `json:"galgameId"`
		User       userObj  `json:"user"`
		TargetUser *userObj `json:"targetUser"`
		LikeCount  int      `json:"likeCount"`
		Created    string   `json:"created"`
	}

	resp := commentResp{
		ID: comment.ID, Content: comment.Content, GalgameID: comment.GalgameID,
		User:      userObj{ID: user.UID, Name: creatorName, Avatar: creatorAvatar},
		LikeCount: 0, Created: comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if req.TargetUserID != nil {
		var targetName, targetAvatar string
		h.db.Table(`"user"`).Select("name, avatar").Where("id = ?", *req.TargetUserID).Row().Scan(&targetName, &targetAvatar)
		resp.TargetUser = &userObj{ID: *req.TargetUserID, Name: targetName, Avatar: targetAvatar}
	}

	return response.OK(c, resp)
}

// DeleteComment deletes a galgame comment.
// DELETE /api/galgame/:gid/comment
func (h *GalgameHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `query:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var comment model.GalgameComment
	if err := h.db.First(&comment, req.CommentID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该评论"))
	}
	if comment.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此评论"))
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("galgame_comment_id = ?", req.CommentID).Delete(&model.GalgameCommentLike{})
		tx.Delete(&comment)
		tx.Model(&model.GalgameLocal{}).Where("id = ?", comment.GalgameID).
			Update("comment_count", gorm.Expr("comment_count - 1"))
		return nil
	})

	return response.OKMessage(c, "评论已删除")
}

// ToggleCommentLike toggles like on a galgame comment.
// PUT /api/galgame/:gid/comment/like
func (h *GalgameHandler) ToggleCommentLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `json:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var comment model.GalgameComment
		tx.First(&comment, req.CommentID)

		var existing model.GalgameCommentLike
		result := tx.Where("user_id = ? AND galgame_comment_id = ?", user.UID, req.CommentID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameCommentLike{UserID: user.UID, CommentID: req.CommentID})
			tx.Model(&model.GalgameComment{}).Where("id = ?", req.CommentID).
				Update("like_count", gorm.Expr("like_count + 1"))
			if comment.UserID != user.UID {
				tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameComment{}).Where("id = ?", req.CommentID).
				Update("like_count", gorm.Expr("like_count - 1"))
			if comment.UserID != user.UID {
				tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
			}
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

func (h *GalgameHandler) getGalgameOwner(c *fiber.Ctx, gid int) int {
	data, err := h.galgameClient.Get(c.Context(), fmt.Sprintf("/galgame/%d", gid), nil)
	if err != nil {
		return 0
	}
	var detail struct {
		Galgame struct {
			UserID int `json:"user_id"`
		} `json:"galgame"`
	}
	json.Unmarshal(data, &detail)
	return detail.Galgame.UserID
}

func createDedupMessage(db *gorm.DB, senderID, receiverID int, msgType string, galgameID int) {
	if senderID == receiverID {
		return
	}
	link := fmt.Sprintf("/galgame/%d", galgameID)
	var count int64
	db.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}
	db.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Link: link, Status: "unread",
	})
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// GetList wraps the wiki /galgame list, renaming "items" → "galgames"
// to match the frontend contract.
// GET /api/galgame
func (h *GalgameHandler) GetList(c *fiber.Ctx) error {
	query := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query.Set(string(key), string(value))
	})

	data, appErr := h.galgameClient.Get(c.Context(), "/galgame", query)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	type wikiGalgame struct {
		ID                 int    `json:"id"`
		NameEnUs           string `json:"name_en_us"`
		NameJaJp           string `json:"name_ja_jp"`
		NameZhCn           string `json:"name_zh_cn"`
		NameZhTw           string `json:"name_zh_tw"`
		Banner             string `json:"banner"`
		ContentLimit       string `json:"content_limit"`
		View               int    `json:"view"`
		ResourceUpdateTime string `json:"resource_update_time"`
		UserID             int    `json:"user_id"`
	}
	var parsed struct {
		Items []wikiGalgame `json:"items"`
		Total int64         `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return response.Error(c, errors.ErrInternal("解析 Wiki 响应失败"))
	}

	// Batch load users and like counts from local DB
	galgameIDs := make([]int, len(parsed.Items))
	userIDs := make([]int, len(parsed.Items))
	for i, g := range parsed.Items {
		galgameIDs[i] = g.ID
		userIDs[i] = g.UserID
	}

	var users []userModel.UserBrief
	if len(userIDs) > 0 {
		h.db.Where("id IN ?", userIDs).Find(&users)
	}
	userMap := make(map[int]userModel.UserBrief, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Like counts
	type likeRow struct {
		GalgameID int `gorm:"column:galgame_id"`
		Count     int `gorm:"column:count"`
	}
	var likes []likeRow
	if len(galgameIDs) > 0 {
		h.db.Table("galgame_like").
			Select("galgame_id, COUNT(*) AS count").
			Where("galgame_id IN ?", galgameIDs).
			Group("galgame_id").Scan(&likes)
	}
	likeMap := make(map[int]int, len(likes))
	for _, l := range likes {
		likeMap[l.GalgameID] = l.Count
	}

	// Resource platforms and languages
	type resRow struct {
		GalgameID int    `gorm:"column:galgame_id"`
		Platform  string `gorm:"column:platform"`
		Language  string `gorm:"column:language"`
	}
	var resources []resRow
	if len(galgameIDs) > 0 {
		h.db.Table("galgame_resource").
			Select("DISTINCT galgame_id, platform, language").
			Where("galgame_id IN ?", galgameIDs).Scan(&resources)
	}
	platformMap := make(map[int][]string)
	languageMap := make(map[int][]string)
	for _, r := range resources {
		if r.Platform != "" {
			platformMap[r.GalgameID] = appendUnique(
				platformMap[r.GalgameID], r.Platform,
			)
		}
		if r.Language != "" {
			languageMap[r.GalgameID] = appendUnique(
				languageMap[r.GalgameID], r.Language,
			)
		}
	}

	galgames := make([]fiber.Map, len(parsed.Items))
	for i, g := range parsed.Items {
		galgames[i] = fiber.Map{
			"id": g.ID,
			"name": fiber.Map{
				"en-us": g.NameEnUs,
				"ja-jp": g.NameJaJp,
				"zh-cn": g.NameZhCn,
				"zh-tw": g.NameZhTw,
			},
			"banner":             g.Banner,
			"user":               userMap[g.UserID],
			"contentLimit":       g.ContentLimit,
			"view":               g.View,
			"likeCount":          likeMap[g.ID],
			"resourceUpdateTime": g.ResourceUpdateTime,
			"platform":           emptyIfNil(platformMap[g.ID]),
			"language":           emptyIfNil(languageMap[g.ID]),
		}
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "成功",
		"data": fiber.Map{
			"galgames": galgames,
			"total":    parsed.Total,
		},
	})
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// toWikiPath converts the Fiber request path to the wiki service path.
// It strips the "/api" prefix and translates frontend route names
// (e.g. "/galgame-tag") to wiki route names (e.g. "/tag").
func toWikiPath(path string) string {
	// Strip /api prefix: "/api/galgame-tag/..." → "/galgame-tag/..."
	wp := path[4:]
	// Translate frontend prefixes to wiki prefixes
	for _, prefix := range []string{
		"/galgame-tag", "/galgame-official",
		"/galgame-engine", "/galgame-series",
		"/galgame-resource",
	} {
		wiki := "/" + prefix[len("/galgame-"):]
		if wp == prefix || len(wp) > len(prefix) && wp[len(prefix)] == '/' {
			return wiki + wp[len(prefix):]
		}
	}
	return wp
}

// ProxyGet forwards a GET request to wiki service (for endpoints with no local side effects).
func (h *GalgameHandler) ProxyGet(c *fiber.Ctx) error {
	wikiPath := toWikiPath(c.Path())

	query := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query.Set(string(key), string(value))
	})

	data, appErr := h.galgameClient.Get(c.Context(), wikiPath, query)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// ProxyWriteWithToken forwards a POST/PUT/DELETE request with OAuth token.
func (h *GalgameHandler) ProxyWriteWithToken(method string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, appErr := middleware.MustGetUser(c)
		if appErr != nil {
			return response.Error(c, appErr)
		}

		token := getAccessToken(c)
		if token == "" {
			return response.Error(c, errors.ErrBadRequest("缺少 OAuth 访问令牌"))
		}

		wikiPath := toWikiPath(c.Path())

		var data json.RawMessage
		switch method {
		case "POST":
			data, appErr = h.galgameClient.PostWithToken(c.Context(), wikiPath, token, json.RawMessage(c.Body()))
		case "PUT":
			data, appErr = h.galgameClient.PutWithToken(c.Context(), wikiPath, token, json.RawMessage(c.Body()))
		case "DELETE":
			data, appErr = h.galgameClient.DeleteWithToken(c.Context(), wikiPath, token, json.RawMessage(c.Body()))
		}
		if appErr != nil {
			return response.Error(c, appErr)
		}

		return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
	}
}

// GetResourceList returns the latest galgame resources.
// GET /api/galgame-resource
func (h *GalgameHandler) GetResourceList(c *fiber.Ctx) error {
	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type resourceRow struct {
		ID        int    `gorm:"column:id"`
		View      int    `gorm:"column:view"`
		GalgameID int    `gorm:"column:galgame_id"`
		UserID    int    `gorm:"column:user_id"`
		Type      string `gorm:"column:type"`
		Language  string `gorm:"column:language"`
		Platform  string `gorm:"column:platform"`
		Size      string `gorm:"column:size"`
		Status    int    `gorm:"column:status"`
		Download  int    `gorm:"column:download"`
		LikeCount int    `gorm:"column:like_count"`
		Note      string `gorm:"column:note"`
		Created   string `gorm:"column:created"`
		Edited    *string `gorm:"column:edited"`
	}

	var total int64
	h.db.Table("galgame_resource").Count(&total)

	offset := (req.Page - 1) * req.Limit
	var rows []resourceRow
	h.db.Table("galgame_resource").
		Order("created DESC").
		Offset(offset).Limit(req.Limit).
		Scan(&rows)

	// Batch load galgame names from wiki
	galgameIDs := make([]int, 0, len(rows))
	userIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		galgameIDs = append(galgameIDs, r.GalgameID)
		userIDs = append(userIDs, r.UserID)
	}

	gnMap, _ := h.galgameClient.GetBatch(c.Context(), galgameIDs)
	if gnMap == nil {
		gnMap = map[int]client.GalgameBrief{}
	}

	var users []userModel.UserBrief
	if len(userIDs) > 0 {
		h.db.Where("id IN ?", userIDs).Find(&users)
	}
	uMap := make(map[int]userModel.UserBrief, len(users))
	for _, u := range users {
		uMap[u.ID] = u
	}

	resources := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		gn := gnMap[r.GalgameID]
		resources = append(resources, fiber.Map{
			"id":         r.ID,
			"view":       r.View,
			"galgameId":  r.GalgameID,
			"user":       uMap[r.UserID],
			"type":       r.Type,
			"language":   r.Language,
			"platform":   r.Platform,
			"size":       r.Size,
			"status":     r.Status,
			"download":   r.Download,
			"likeCount":  r.LikeCount,
			"isLiked":    false,
			"linkDomain": "",
			"note":       r.Note,
			"created":    r.Created,
			"edited":     r.Edited,
			"galgameName": fiber.Map{
				"en-us": gn.NameEnUs,
				"ja-jp": gn.NameJaJp,
				"zh-cn": gn.NameZhCn,
				"zh-tw": gn.NameZhTw,
			},
		})
	}

	return response.OK(c, fiber.Map{
		"resources": resources,
		"total":     total,
	})
}

// GetAllRatings returns paginated galgame ratings.
// GET /api/galgame-rating/all
func (h *GalgameHandler) GetAllRatings(c *fiber.Ctx) error {
	var req struct {
		Page         int    `query:"page" validate:"min=1"`
		Limit        int    `query:"limit" validate:"min=1,max=50"`
		SortField    string `query:"sortField"`
		SortOrder    string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
		SpoilerLevel string `query:"spoilerLevel"`
		PlayStatus   string `query:"playStatus"`
		GalgameType  string `query:"galgameType"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	query := h.db.Table("galgame_rating r")
	if req.SpoilerLevel != "" && req.SpoilerLevel != "all" {
		query = query.Where("r.spoiler_level = ?", req.SpoilerLevel)
	}
	if req.PlayStatus != "" && req.PlayStatus != "all" {
		query = query.Where("r.play_status = ?", req.PlayStatus)
	}
	if req.GalgameType != "" && req.GalgameType != "all" {
		query = query.Where("r.galgame_type @> ?", fmt.Sprintf(`["%s"]`, req.GalgameType))
	}

	var total int64
	query.Count(&total)

	orderCol := "r.created"
	switch req.SortField {
	case "view":
		orderCol = "r.view"
	case "overall":
		orderCol = "r.overall"
	}

	type ratingRow struct {
		ID           int     `gorm:"column:id"`
		Recommend    string  `gorm:"column:recommend"`
		Overall      int     `gorm:"column:overall"`
		View         int     `gorm:"column:view"`
		GalgameType  string  `gorm:"column:galgame_type"`
		PlayStatus   string  `gorm:"column:play_status"`
		ShortSummary string  `gorm:"column:short_summary"`
		SpoilerLevel string  `gorm:"column:spoiler_level"`
		Art          int     `gorm:"column:art"`
		Story        int     `gorm:"column:story"`
		Music        int     `gorm:"column:music"`
		Character    int     `gorm:"column:character"`
		Route        int     `gorm:"column:route"`
		System       int     `gorm:"column:system"`
		Voice        int     `gorm:"column:voice"`
		ReplayValue  int     `gorm:"column:replay_value"`
		LikeCount    int     `gorm:"column:like_count"`
		Created      string  `gorm:"column:created"`
		Updated      string  `gorm:"column:updated"`
		UserID       int     `gorm:"column:user_id"`
		GalgameID    int     `gorm:"column:galgame_id"`
	}
	var rows []ratingRow
	query.Select("r.*").
		Order(orderCol + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Scan(&rows)

	// Batch load users and galgames
	userIDs := make([]int, len(rows))
	galgameIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		galgameIDs[i] = r.GalgameID
	}

	var users []userModel.UserBrief
	if len(userIDs) > 0 {
		h.db.Where("id IN ?", userIDs).Find(&users)
	}
	userMap := make(map[int]userModel.UserBrief, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	gMap, _ := h.galgameClient.GetBatch(c.Context(), galgameIDs)
	if gMap == nil {
		gMap = map[int]client.GalgameBrief{}
	}

	ratingData := make([]fiber.Map, len(rows))
	for i, r := range rows {
		g := gMap[r.GalgameID]
		ratingData[i] = fiber.Map{
			"id":            r.ID,
			"user":          userMap[r.UserID],
			"recommend":     r.Recommend,
			"overall":       r.Overall,
			"view":          r.View,
			"galgameType":   json.RawMessage(r.GalgameType),
			"play_status":   r.PlayStatus,
			"short_summary": r.ShortSummary,
			"spoiler_level": r.SpoilerLevel,
			"art":           r.Art,
			"story":         r.Story,
			"music":         r.Music,
			"character":     r.Character,
			"route":         r.Route,
			"system":        r.System,
			"voice":         r.Voice,
			"replay_value":  r.ReplayValue,
			"likeCount":     r.LikeCount,
			"created":       r.Created,
			"updated":       r.Updated,
			"galgame": fiber.Map{
				"id":           g.ID,
				"contentLimit": g.ContentLimit,
				"name": fiber.Map{
					"en-us": g.NameEnUs,
					"ja-jp": g.NameJaJp,
					"zh-cn": g.NameZhCn,
					"zh-tw": g.NameZhTw,
				},
			},
		}
	}

	return response.OK(c, fiber.Map{
		"ratingData": ratingData,
		"total":      total,
	})
}

// GetSeriesList wraps wiki /series, transforming response for frontend.
// GET /api/galgame-series
func (h *GalgameHandler) GetSeriesList(c *fiber.Ctx) error {
	query := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query.Set(string(key), string(value))
	})

	data, appErr := h.galgameClient.Get(c.Context(), "/series", query)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	type wikiGalgame struct {
		NameEnUs      string `json:"name_en_us"`
		NameJaJp      string `json:"name_ja_jp"`
		NameZhCn      string `json:"name_zh_cn"`
		NameZhTw      string `json:"name_zh_tw"`
		Banner        string `json:"banner"`
		ContentLimit  string `json:"content_limit"`
	}
	type wikiSeries struct {
		ID          int           `json:"id"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Galgame     []wikiGalgame `json:"galgame"`
		Created     string        `json:"created"`
		Updated     string        `json:"updated"`
	}
	var parsed struct {
		Items []wikiSeries `json:"items"`
		Total int64        `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return response.Error(c, errors.ErrInternal("解析 Wiki 响应失败"))
	}

	series := make([]fiber.Map, len(parsed.Items))
	for i, s := range parsed.Items {
		isNSFW := false
		samples := make([]fiber.Map, 0, min(len(s.Galgame), 4))
		for j, g := range s.Galgame {
			if g.ContentLimit == "nsfw" {
				isNSFW = true
			}
			if j < 4 {
				samples = append(samples, fiber.Map{
					"name": fiber.Map{
						"en-us": g.NameEnUs,
						"ja-jp": g.NameJaJp,
						"zh-cn": g.NameZhCn,
						"zh-tw": g.NameZhTw,
					},
					"banner": g.Banner,
				})
			}
		}
		series[i] = fiber.Map{
			"id":             s.ID,
			"name":           s.Name,
			"description":    s.Description,
			"isNSFW":         isNSFW,
			"sampleGalgame":  samples,
			"galgameCount":   len(s.Galgame),
			"created":        s.Created,
			"updated":        s.Updated,
		}
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "成功",
		"data": fiber.Map{
			"series": series,
			"total":  parsed.Total,
		},
	})
}

// GetOfficialList wraps wiki /official, transforming for frontend.
// GET /api/galgame-official
func (h *GalgameHandler) GetOfficialList(c *fiber.Ctx) error {
	query := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query.Set(string(key), string(value))
	})

	data, appErr := h.galgameClient.Get(c.Context(), "/official", query)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	type wikiAlias struct {
		Name string `json:"name"`
	}
	type wikiOfficial struct {
		ID          int         `json:"id"`
		Name        string      `json:"name"`
		Link        string      `json:"link"`
		Category    string      `json:"category"`
		Lang        string      `json:"lang"`
		Alias       []wikiAlias `json:"alias"`
		GalgameCount int        `json:"galgame_count"`
	}
	var parsed struct {
		Items []wikiOfficial `json:"items"`
		Total int64          `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return response.Error(c, errors.ErrInternal("解析 Wiki 响应失败"))
	}

	officials := make([]fiber.Map, len(parsed.Items))
	for i, o := range parsed.Items {
		aliasNames := make([]string, len(o.Alias))
		for j, a := range o.Alias {
			aliasNames[j] = a.Name
		}
		officials[i] = fiber.Map{
			"id":           o.ID,
			"name":         o.Name,
			"link":         o.Link,
			"category":     o.Category,
			"lang":         o.Lang,
			"alias":        aliasNames,
			"galgameCount": o.GalgameCount,
		}
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "成功",
		"data": fiber.Map{
			"officials": officials,
			"total":     parsed.Total,
		},
	})
}

// GetTagList wraps wiki /tag, transforming for frontend.
// GET /api/galgame-tag
func (h *GalgameHandler) GetTagList(c *fiber.Ctx) error {
	query := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		query.Set(string(key), string(value))
	})

	data, appErr := h.galgameClient.Get(c.Context(), "/tag", query)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	type wikiTag struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Category     string `json:"category"`
		GalgameCount int    `json:"galgame_count"`
	}
	var parsed struct {
		Items []wikiTag `json:"items"`
		Total int64     `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return response.Error(c, errors.ErrInternal("解析 Wiki 响应失败"))
	}

	tags := make([]fiber.Map, len(parsed.Items))
	for i, t := range parsed.Items {
		tags[i] = fiber.Map{
			"id":           t.ID,
			"name":         t.Name,
			"category":     t.Category,
			"galgameCount": t.GalgameCount,
		}
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "成功",
		"data": fiber.Map{
			"tags": tags,
			"total": parsed.Total,
		},
	})
}
