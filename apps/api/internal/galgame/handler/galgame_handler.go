package handler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
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
			tx.Create(&model.GalgameStats{GalgameID: created.ID})
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

	// Fetch wiki metadata
	wikiData, appErr := h.galgameClient.Get(c.Context(), "/galgame/"+gid, nil)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gidInt, _ := strconv.Atoi(gid)

	// Fetch local stats
	var stats model.GalgameStats
	h.db.Where("galgame_id = ?", gidInt).FirstOrCreate(&stats, model.GalgameStats{GalgameID: gidInt})

	// Fetch user interaction
	isLiked := false
	isFavorited := false
	if userInfo != nil {
		var likeCount, favCount int64
		h.db.Model(&model.GalgameLike{}).Where("user_id = ? AND galgame_id = ?", userInfo.UID, gidInt).Count(&likeCount)
		h.db.Model(&model.GalgameFavorite{}).Where("user_id = ? AND galgame_id = ?", userInfo.UID, gidInt).Count(&favCount)
		isLiked = likeCount > 0
		isFavorited = favCount > 0
	}

	// Merge response
	var wikiResult json.RawMessage = wikiData
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "成功",
		"data": fiber.Map{
			"wiki":        wikiResult,
			"stats":       stats,
			"isLiked":     isLiked,
			"isFavorited": isFavorited,
		},
	})
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
			tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", gid).
				Update("like_count", gorm.Expr("like_count + 1"))
			if ownerID > 0 {
				tx.Model(&userModel.User{}).Where("id = ?", ownerID).
					Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
				createDedupMessage(tx, user.UID, ownerID, "liked", gid)
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", gid).
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
			tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", gid).
				Update("favorite_count", gorm.Expr("favorite_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", gid).
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

	type commentRow struct {
		ID             int    `json:"id"`
		Content        string `json:"content"`
		GalgameID      int    `json:"galgame_id"`
		UserID         int    `json:"user_id"`
		UserName       string `json:"userName"`
		UserAvatar     string `json:"userAvatar"`
		TargetUserID   *int   `json:"target_user_id"`
		TargetUserName string `json:"targetUserName"`
		LikeCount      int    `json:"like_count"`
		CreatedAt      string `json:"created"`
	}

	var comments []commentRow
	var total int64

	h.db.Model(&model.GalgameComment{}).Where("galgame_id = ?", gid).Count(&total)
	h.db.Table("galgame_comment gc").
		Select(`gc.id, gc.content, gc.galgame_id, gc.user_id,
			u1.name AS user_name, u1.avatar AS user_avatar,
			gc.target_user_id, u2.name AS target_user_name,
			gc.like_count, gc.created AS created_at`).
		Joins(`LEFT JOIN "user" u1 ON u1.id = gc.user_id`).
		Joins(`LEFT JOIN "user" u2 ON u2.id = gc.target_user_id`).
		Where("gc.galgame_id = ?", gid).
		Order("gc.created DESC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&comments)

	return response.Paginated(c, comments, total)
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
		tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", gid).
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

	return response.OK(c, comment)
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
		tx.Model(&model.GalgameStats{}).Where("galgame_id = ?", comment.GalgameID).
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

// ProxyGet forwards a GET request to wiki service (for endpoints with no local side effects).
func (h *GalgameHandler) ProxyGet(c *fiber.Ctx) error {
	path := c.Path()
	// Strip /api prefix to get wiki path
	wikiPath := path[4:] // "/api/galgame/..." → "/galgame/..."

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

// ProxyPostWithToken forwards a POST/PUT/DELETE request with OAuth token.
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

		wikiPath := c.Path()[4:]

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
