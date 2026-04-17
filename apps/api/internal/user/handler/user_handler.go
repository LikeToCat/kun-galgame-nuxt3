package handler

import (
	"encoding/json"
	"strconv"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserHandler struct {
	db          *gorm.DB
	userService *service.UserService
	wikiGC      *galgameClient.GalgameClient
}

func NewUserHandler(db *gorm.DB, userService *service.UserService, gc *galgameClient.GalgameClient) *UserHandler {
	return &UserHandler{db: db, userService: userService, wikiGC: gc}
}

// GetProfile returns a user's public profile.
// GET /api/user/:uid
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	profile, appErr := h.userService.GetUserProfile(c.Context(), uid)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Enrich with wiki galgame stats (non-blocking — zero values on failure)
	if wikiStats, err := h.wikiGC.GetUserStats(c.Context(), uid); err == nil && wikiStats != nil {
		profile.Galgame = wikiStats.GalgameCreated
		profile.DailyGalgameCount = wikiStats.GalgameCreatedToday
		profile.ContributeGalgame = wikiStats.GalgameContributed
	}

	return response.OK(c, profile)
}

// CheckIn handles daily check-in.
// POST /api/user/check-in
func (h *UserHandler) CheckIn(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	points, appErr := h.userService.CheckIn(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, points)
}

// UpdateBio updates the user's bio.
// PUT /api/user/bio
func (h *UserHandler) UpdateBio(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateBioRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateBio(c.Context(), user.UID, req.Bio); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "签名更新成功")
}

// UpdateUsername updates the user's name (costs moemoepoints).
// PUT /api/user/username
func (h *UserHandler) UpdateUsername(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateUsernameRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateUsername(c.Context(), user.UID, req.Username); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户名更新成功")
}

// UpdateEmail updates the user's email after code verification.
// PUT /api/user/email
func (h *UserHandler) UpdateEmail(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateEmailRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateEmail(c.Context(), user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "邮箱更新成功")
}

// GetEmail returns the user's masked email.
// GET /api/user/email
func (h *UserHandler) GetEmail(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	email, appErr := h.userService.GetMaskedEmail(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, email)
}

// GetStatus returns the user's status (moemoepoints, check-in, unread messages).
// GET /api/user/status
func (h *UserHandler) GetStatus(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	status, appErr := h.userService.GetUserStatus(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, status)
}

// UploadAvatar handles avatar upload.
// POST /api/user/avatar
func (h *UserHandler) UploadAvatar(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}

	link, appErr := h.userService.UploadAvatar(c.Context(), user.UID, buf)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, link)
}

// GetUserGalgames returns a user's galgame list.
// GET /api/user/:uid/galgames
func (h *UserHandler) GetUserGalgames(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserGalgamesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	ids, total, appErr := h.userService.GetUserGalgameIDs(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if len(ids) == 0 {
		return response.Paginated(c, []fiber.Map{}, total)
	}

	briefMap, wikiErr := h.wikiGC.GetBatch(c.Context(), ids)
	if wikiErr != nil {
		return response.Paginated(c, []fiber.Map{}, total)
	}

	// Local counts (view, like_count)
	type localRow struct {
		ID        int `gorm:"column:id"`
		View      int `gorm:"column:view"`
		LikeCount int `gorm:"column:like_count"`
	}
	var locals []localRow
	h.db.Table("galgame").Select("id, view, like_count").
		Where("id IN ?", ids).Scan(&locals)
	localMap := make(map[int]localRow, len(locals))
	for _, l := range locals {
		localMap[l.ID] = l
	}

	// Resource platforms & languages
	type resRow struct {
		GalgameID int    `gorm:"column:galgame_id"`
		Platform  string `gorm:"column:platform"`
		Language  string `gorm:"column:language"`
	}
	var resources []resRow
	h.db.Table("galgame_resource").
		Select("DISTINCT galgame_id, platform, language").
		Where("galgame_id IN ?", ids).Scan(&resources)

	platformMap := make(map[int][]string)
	languageMap := make(map[int][]string)
	for _, r := range resources {
		if r.Platform != "" {
			platformMap[r.GalgameID] = appendUniqueStr(platformMap[r.GalgameID], r.Platform)
		}
		if r.Language != "" {
			languageMap[r.GalgameID] = appendUniqueStr(languageMap[r.GalgameID], r.Language)
		}
	}

	// User lookup from wiki briefs
	userIDs := make([]int, 0)
	seen := map[int]bool{}
	for _, b := range briefMap {
		if b.UserID > 0 && !seen[b.UserID] {
			userIDs = append(userIDs, b.UserID)
			seen[b.UserID] = true
		}
	}
	type userBrief struct {
		ID     int    `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Avatar string `gorm:"column:avatar"`
	}
	var users []userBrief
	if len(userIDs) > 0 {
		h.db.Table(`"user"`).Select("id, name, avatar").Where("id IN ?", userIDs).Scan(&users)
	}
	userMap := make(map[int]userBrief, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	items := make([]fiber.Map, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		l := localMap[id]
		u := userMap[b.UserID]
		items = append(items, fiber.Map{
			"id": b.ID,
			"name": fiber.Map{
				"en-us": b.NameEnUs, "ja-jp": b.NameJaJp,
				"zh-cn": b.NameZhCn, "zh-tw": b.NameZhTw,
			},
			"banner":             b.Banner,
			"user":               fiber.Map{"id": u.ID, "name": u.Name, "avatar": u.Avatar},
			"contentLimit":       b.ContentLimit,
			"view":               l.View,
			"likeCount":          l.LikeCount,
			"resourceUpdateTime": b.ResourceUpdateTime,
			"platform":           emptyStrSlice(platformMap[id]),
			"language":           emptyStrSlice(languageMap[id]),
		})
	}

	return response.Paginated(c, items, total)
}

// GetUserTopics returns a user's topic list.
// GET /api/user/:uid/topics
func (h *UserHandler) GetUserTopics(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, total, appErr := h.userService.GetUserTopics(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, fiber.Map{"topics": items, "total": total})
}

// GetUserReplies returns a user's reply list.
// GET /api/user/:uid/replies
func (h *UserHandler) GetUserReplies(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserRepliesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, total, appErr := h.userService.GetUserReplies(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, fiber.Map{"replies": items, "total": total})
}

// GetUserComments returns a user's comment list.
// GET /api/user/:uid/comments
func (h *UserHandler) GetUserComments(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserCommentsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, total, appErr := h.userService.GetUserComments(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, fiber.Map{"comments": items, "total": total})
}

// GetUserResources returns a user's galgame resource list.
// GET /api/user/:uid/resources
func (h *UserHandler) GetUserResources(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserResourcesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	rows, total, appErr := h.userService.GetUserResources(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Collect resource IDs for link lookup and galgame IDs for wiki batch
	resourceIDs := make([]int, len(rows))
	galgameIDs := make([]int, 0, len(rows))
	seen := map[int]bool{}
	for i, r := range rows {
		resourceIDs[i] = r.ID
		if !seen[r.GalgameID] {
			galgameIDs = append(galgameIDs, r.GalgameID)
			seen[r.GalgameID] = true
		}
	}

	// Batch load links from galgame_resource_link table
	linkMap := make(map[int][]string)
	if len(resourceIDs) > 0 {
		linkMap, _ = h.userService.GetResourceLinks(resourceIDs)
	}

	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = h.wikiGC.GetBatch(c.Context(), galgameIDs)
	}

	resources := make([]fiber.Map, len(rows))
	for i, r := range rows {
		links := linkMap[r.ID]
		if links == nil {
			links = []string{}
		}

		galgameName := fiber.Map{"en-us": "", "ja-jp": "", "zh-cn": "", "zh-tw": ""}
		if briefMap != nil {
			if b, ok := briefMap[r.GalgameID]; ok {
				galgameName = fiber.Map{
					"en-us": b.NameEnUs, "ja-jp": b.NameJaJp,
					"zh-cn": b.NameZhCn, "zh-tw": b.NameZhTw,
				}
			}
		}

		resources[i] = fiber.Map{
			"id":          r.ID,
			"galgameId":   r.GalgameID,
			"galgameName": galgameName,
			"type":        r.Type,
			"language":    r.Language,
			"platform":    r.Platform,
			"size":        r.Size,
			"link":        links,
			"code":        r.Code,
			"password":    r.Password,
			"note":        r.Note,
			"status":      r.Status,
			"created":     r.Created,
		}
	}

	return response.OK(c, fiber.Map{"resources": resources, "total": total})
}

// GetUserRatings returns a user's galgame rating list.
// GET /api/user/:uid/ratings
func (h *UserHandler) GetUserRatings(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserRatingsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	rows, total, appErr := h.userService.GetUserRatings(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Collect galgame IDs for wiki batch lookup
	galgameIDs := make([]int, 0, len(rows))
	seen := map[int]bool{}
	for _, r := range rows {
		if !seen[r.GalgameID] {
			galgameIDs = append(galgameIDs, r.GalgameID)
			seen[r.GalgameID] = true
		}
	}

	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = h.wikiGC.GetBatch(c.Context(), galgameIDs)
	}

	ratingData := make([]fiber.Map, len(rows))
	for i, r := range rows {
		var galgameType []string
		if r.GalgameType != "" {
			_ = json.Unmarshal([]byte(r.GalgameType), &galgameType)
		}

		galgameInfo := fiber.Map{"id": r.GalgameID, "name": fiber.Map{"en-us": "", "ja-jp": "", "zh-cn": "", "zh-tw": ""}, "contentLimit": ""}
		if briefMap != nil {
			if b, ok := briefMap[r.GalgameID]; ok {
				galgameInfo = fiber.Map{
					"id": b.ID,
					"name": fiber.Map{
						"en-us": b.NameEnUs, "ja-jp": b.NameJaJp,
						"zh-cn": b.NameZhCn, "zh-tw": b.NameZhTw,
					},
					"contentLimit": b.ContentLimit,
				}
			}
		}

		ratingData[i] = fiber.Map{
			"id":             r.ID,
			"user":           fiber.Map{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvatar},
			"recommend":      r.Recommend,
			"overall":        r.Overall,
			"view":           r.View,
			"galgameType":    galgameType,
			"play_status":    r.PlayStatus,
			"art":            r.Art,
			"story":          r.Story,
			"music":          r.Music,
			"character":      r.Character,
			"route":          r.Route,
			"system":         r.System,
			"voice":          r.Voice,
			"replay_value":   r.ReplayValue,
			"spoiler_level":  r.SpoilerLevel,
			"likeCount":      r.LikeCount,
			"created":        r.Created,
			"updated":        r.Updated,
			"galgame":        galgameInfo,
		}
	}

	return response.OK(c, fiber.Map{"ratingData": ratingData, "total": total})
}

// BanUser bans or unbans a user (admin only).
// PUT /api/user/:uid/ban
func (h *UserHandler) BanUser(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.BanUserRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.userService.BanUser(c.Context(), uid, req.Status); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户状态更新成功")
}

// DeleteUser permanently deletes a user (admin only).
// DELETE /api/user/:uid
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	if appErr := h.userService.DeleteUser(c.Context(), uid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户已删除")
}

// GetFloatingCard returns lightweight user info for hover card.
// GET /api/user/:uid/floating
func (h *UserHandler) GetFloatingCard(c *fiber.Ctx) error {
	var req struct {
		UserID int `query:"userId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type userRow struct {
		ID           int    `gorm:"column:id"`
		Name         string `gorm:"column:name"`
		Avatar       string `gorm:"column:avatar"`
		Moemoepoint  int    `gorm:"column:moemoepoint"`
		Bio          string `gorm:"column:bio"`
		Status       int    `gorm:"column:status"`
	}
	var user userRow
	if err := h.db.Table(`"user"`).Where("id = ?", req.UserID).Scan(&user).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该用户"))
	}
	if user.Status == 1 {
		return response.Error(c, errors.ErrNotFound("该用户已被封禁"))
	}

	// Count various contributions in a single query
	type floatingStats struct {
		TopicCount        int64 `gorm:"column:topic_count"`
		TopicReplyCount   int64 `gorm:"column:topic_reply_count"`
		TopicCommentCount int64 `gorm:"column:topic_comment_count"`
		ResourceCount     int64 `gorm:"column:resource_count"`
	}
	var stats floatingStats
	h.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM topic WHERE user_id = @uid) AS topic_count,
			(SELECT COUNT(*) FROM topic_reply WHERE user_id = @uid) AS topic_reply_count,
			(SELECT COUNT(*) FROM topic_comment WHERE user_id = @uid)
				+ (SELECT COUNT(*) FROM galgame_comment WHERE user_id = @uid)
				+ (SELECT COUNT(*) FROM galgame_website_comment WHERE user_id = @uid) AS topic_comment_count,
			(SELECT COUNT(*) FROM galgame_resource WHERE user_id = @uid) AS resource_count
	`, map[string]any{"uid": req.UserID}).Scan(&stats)

	return response.OK(c, fiber.Map{
		"id":                   user.ID,
		"name":                 user.Name,
		"avatar":               user.Avatar,
		"moemoepoint":          user.Moemoepoint,
		"topicCount":           stats.TopicCount,
		"topicReplyCount":      stats.TopicReplyCount,
		"topicCommentCount":    stats.TopicCommentCount,
		"galgameResourceCount": stats.ResourceCount,
	})
}

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func emptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
