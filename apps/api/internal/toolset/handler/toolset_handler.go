package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────
// Handler
// ──────────────────────────────────────────

type ToolsetHandler struct {
	db  *gorm.DB
	s3  *storage.S3Client
	rdb *redis.Client
}

func NewToolsetHandler(db *gorm.DB, s3 *storage.S3Client, rdb *redis.Client) *ToolsetHandler {
	return &ToolsetHandler{db: db, s3: s3, rdb: rdb}
}

// ══════════════════════════════════════════
// CRUD
// ══════════════════════════════════════════

// GetList returns a paginated list of toolsets with filters.
// GET /api/toolset
func (h *ToolsetHandler) GetList(c *fiber.Ctx) error {
	var req struct {
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=100"`
		Type      string `query:"type"`
		Language  string `query:"language"`
		Platform  string `query:"platform"`
		Version   string `query:"version"`
		SortField string `query:"sortField"`
		SortOrder string `query:"sortOrder"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 24
	}

	query := h.db.Model(&model.GalgameToolset{}).Where("status != 1")

	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Language != "" {
		query = query.Where("language = ?", req.Language)
	}
	if req.Platform != "" {
		query = query.Where("platform = ?", req.Platform)
	}
	if req.Version != "" {
		query = query.Where("version = ?", req.Version)
	}

	var total int64
	query.Count(&total)

	// Sort
	sortField := "created"
	if req.SortField != "" {
		allowed := map[string]bool{
			"created": true, "view": true, "name": true,
			"resource_update_time": true,
		}
		if allowed[req.SortField] {
			sortField = req.SortField
		}
	}
	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	var toolsets []model.GalgameToolset
	offset := (req.Page - 1) * req.Limit
	query.Order(sortField + " " + sortOrder).Offset(offset).Limit(req.Limit).Find(&toolsets)

	// Batch load practicality averages
	toolsetIDs := make([]int, len(toolsets))
	for i, t := range toolsets {
		toolsetIDs[i] = t.ID
	}

	type avgRow struct {
		ToolsetID int
		Avg       float64
	}
	var avgs []avgRow
	h.db.Model(&model.GalgameToolsetPracticality{}).
		Select("toolset_id, COALESCE(AVG(rate), 0) AS avg").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").Scan(&avgs)
	avgMap := make(map[int]float64, len(avgs))
	for _, a := range avgs {
		avgMap[a.ToolsetID] = math.Round(a.Avg*100) / 100
	}

	// Batch load download sums
	type dlRow struct {
		ToolsetID int
		Total     int
	}
	var dls []dlRow
	h.db.Model(&model.GalgameToolsetResource{}).
		Select("toolset_id, COALESCE(SUM(download), 0) AS total").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").Scan(&dls)
	dlMap := make(map[int]int, len(dls))
	for _, d := range dls {
		dlMap[d.ToolsetID] = d.Total
	}

	// Batch load comment counts
	type ccRow struct {
		ToolsetID int
		Count     int
	}
	var ccs []ccRow
	h.db.Model(&model.GalgameToolsetComment{}).
		Select("toolset_id, COUNT(*) AS count").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").Scan(&ccs)
	ccMap := make(map[int]int, len(ccs))
	for _, cc := range ccs {
		ccMap[cc.ToolsetID] = cc.Count
	}

	// Batch load users
	userIDs := make([]int, len(toolsets))
	for i, t := range toolsets {
		userIDs[i] = t.UserID
	}
	var users []userModel.UserBrief
	h.db.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[int]userModel.UserBrief, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Build flat ToolsetCard items
	items := make([]fiber.Map, 0, len(toolsets))
	for _, t := range toolsets {
		var practicalityAvg any = avgMap[t.ID]
		if avgMap[t.ID] == 0 {
			practicalityAvg = nil
		}
		items = append(items, fiber.Map{
			"id":                   t.ID,
			"name":                 t.Name,
			"user":                 userMap[t.UserID],
			"type":                 t.Type,
			"platform":             t.Platform,
			"language":             t.Language,
			"version":              t.Version,
			"view":                 t.View,
			"download":             dlMap[t.ID],
			"commentCount":         ccMap[t.ID],
			"practicalityAvg":      practicalityAvg,
			"resource_update_time": t.ResourceUpdateTime,
		})
	}

	return response.Paginated(c, items, total)
}

// Create creates a new toolset.
// POST /api/toolset
func (h *ToolsetHandler) Create(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Name        string   `json:"name" validate:"required,max=500"`
		Description string   `json:"description" validate:"max=2000"`
		Type        string   `json:"type"`
		Language    string   `json:"language"`
		Platform    string   `json:"platform"`
		Homepage    []string `json:"homepage"`
		Version     string   `json:"version" validate:"max=233"`
		Aliases     []string `json:"aliases"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	homepageJSON, _ := json.Marshal(req.Homepage)

	var toolset model.GalgameToolset
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		toolset = model.GalgameToolset{
			Name:        req.Name,
			Description: req.Description,
			Type:        req.Type,
			Language:    req.Language,
			Platform:    req.Platform,
			Homepage:    homepageJSON,
			Version:     req.Version,
			UserID:      user.UID,
		}
		if err := tx.Create(&toolset).Error; err != nil {
			return err
		}

		// Create aliases
		for _, alias := range req.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			tx.Create(&model.GalgameToolsetAlias{
				Name:      alias,
				ToolsetID: toolset.ID,
			})
		}

		// Add creator as contributor
		tx.Create(&model.GalgameToolsetContributor{
			ToolsetID: toolset.ID,
			UserID:    user.UID,
		})

		// Moemoepoint +3
		tx.Model(&userModel.User{}).Where("id = ?", user.UID).
			Update("moemoepoint", gorm.Expr("moemoepoint + ?", 3))

		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("创建工具失败"))
	}

	return response.OK(c, toolset)
}

// GetDetail returns toolset detail.
// GET /api/toolset/:id
func (h *ToolsetHandler) GetDetail(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var toolset model.GalgameToolset
	if err := h.db.First(&toolset, id).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该工具"))
	}

	// View +1 async
	go h.db.Model(&model.GalgameToolset{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))

	// Render description as HTML
	descriptionHTML := markdown.Render(toolset.Description)

	// Aliases
	var aliases []model.GalgameToolsetAlias
	h.db.Where("toolset_id = ?", id).Find(&aliases)

	// User
	var user userModel.UserBrief
	h.db.Where("id = ?", toolset.UserID).First(&user)

	// Practicality distribution
	type rateCount struct {
		Rate  int   `json:"rate"`
		Count int64 `json:"count"`
	}
	var rateCounts []rateCount
	h.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", id).
		Select("rate, COUNT(*) as count").
		Group("rate").
		Scan(&rateCounts)

	counts := map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for _, rc := range rateCounts {
		counts[rc.Rate] = rc.Count
	}

	var avgRate float64
	h.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", id).
		Select("COALESCE(AVG(rate), 0)").Scan(&avgRate)

	// Download sum
	var downloadSum int64
	h.db.Model(&model.GalgameToolsetResource{}).
		Where("toolset_id = ?", id).
		Select("COALESCE(SUM(download), 0)").Scan(&downloadSum)

	// Latest 5 comments
	type commentItem struct {
		model.GalgameToolsetComment
		User userModel.UserBrief `json:"user"`
	}
	var rawComments []model.GalgameToolsetComment
	h.db.Where("toolset_id = ?", id).Order("created DESC").Limit(5).Find(&rawComments)

	comments := make([]commentItem, 0, len(rawComments))
	for _, cm := range rawComments {
		var u userModel.UserBrief
		h.db.Where("id = ?", cm.UserID).First(&u)
		comments = append(comments, commentItem{
			GalgameToolsetComment: cm,
			User:                  u,
		})
	}

	// Contributors
	var contributors []userModel.UserBrief
	h.db.Raw(`SELECT u.id, u.name, u.avatar FROM "user" u
		INNER JOIN galgame_toolset_contributor c ON c.user_id = u.id
		WHERE c.toolset_id = ?`, id).Scan(&contributors)

	// Resources
	var resources []model.GalgameToolsetResource
	h.db.Where("toolset_id = ?", id).Order("created DESC").Find(&resources)

	return response.OK(c, fiber.Map{
		"toolset":         toolset,
		"descriptionHTML": descriptionHTML,
		"aliases":         aliases,
		"user":            user,
		"practicality": fiber.Map{
			"counts": counts,
			"avg":    math.Round(avgRate*100) / 100,
		},
		"downloadSum":  downloadSum,
		"comments":     comments,
		"contributors": contributors,
		"resources":    resources,
	})
}

// Update updates a toolset.
// PUT /api/toolset/:id
func (h *ToolsetHandler) Update(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var toolset model.GalgameToolset
	if err := h.db.First(&toolset, id).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该工具"))
	}

	if toolset.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限编辑此工具"))
	}

	var req struct {
		Name        string   `json:"name" validate:"required,max=500"`
		Description string   `json:"description" validate:"max=2000"`
		Type        string   `json:"type"`
		Language    string   `json:"language"`
		Platform    string   `json:"platform"`
		Homepage    []string `json:"homepage"`
		Version     string   `json:"version" validate:"max=233"`
		Aliases     []string `json:"aliases"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	homepageJSON, _ := json.Marshal(req.Homepage)
	now := time.Now()

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.GalgameToolset{}).Where("id = ?", id).Updates(map[string]any{
			"name":        req.Name,
			"description": req.Description,
			"type":        req.Type,
			"language":    req.Language,
			"platform":    req.Platform,
			"homepage":    homepageJSON,
			"version":     req.Version,
			"edited":      now,
		})

		// Replace aliases
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetAlias{})
		for _, alias := range req.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			tx.Create(&model.GalgameToolsetAlias{
				Name:      alias,
				ToolsetID: id,
			})
		}

		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("更新工具失败"))
	}

	return response.OKMessage(c, "工具更新成功")
}

// Delete deletes a toolset.
// DELETE /api/toolset/:id
func (h *ToolsetHandler) Delete(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var toolset model.GalgameToolset
	if err := h.db.First(&toolset, id).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该工具"))
	}

	if toolset.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此工具"))
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		// S3 cleanup: delete all s3 resources
		var resources []model.GalgameToolsetResource
		tx.Where("toolset_id = ? AND type = 's3'", id).Find(&resources)
		for _, r := range resources {
			if r.Code != "" {
				if err := h.s3.Delete(context.Background(), r.Code); err != nil {
					slog.Warn("删除 S3 资源失败", "key", r.Code, "error", err)
				}
			}
		}

		// Delete related records
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetAlias{})
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetContributor{})
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetPracticality{})
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetResource{})
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetComment{})
		tx.Where("toolset_id = ?", id).Delete(&model.GalgameToolsetCategoryRelation{})

		// Delete toolset itself
		tx.Delete(&model.GalgameToolset{}, id)

		// Moemoepoint deduction
		tx.Model(&userModel.User{}).Where("id = ?", toolset.UserID).
			Update("moemoepoint", gorm.Expr("moemoepoint - ?", 3))

		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("删除工具失败"))
	}

	return response.OKMessage(c, "工具已删除")
}

// ══════════════════════════════════════════
// Practicality
// ══════════════════════════════════════════

// GetPracticality returns rating distribution for a toolset.
// GET /api/toolset/:id/practicality
func (h *ToolsetHandler) GetPracticality(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	type rateCount struct {
		Rate  int   `json:"rate"`
		Count int64 `json:"count"`
	}
	var rateCounts []rateCount
	h.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", id).
		Select("rate, COUNT(*) as count").
		Group("rate").
		Scan(&rateCounts)

	counts := map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for _, rc := range rateCounts {
		counts[rc.Rate] = rc.Count
	}

	var avgRate float64
	h.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", id).
		Select("COALESCE(AVG(rate), 0)").Scan(&avgRate)

	// Current user's rating
	var mine *int
	userInfo := middleware.GetUser(c)
	if userInfo != nil {
		var p model.GalgameToolsetPracticality
		if err := h.db.Where("toolset_id = ? AND user_id = ?", id, userInfo.UID).First(&p).Error; err == nil {
			mine = &p.Rate
		}
	}

	return response.OK(c, fiber.Map{
		"counts": counts,
		"avg":    math.Round(avgRate*100) / 100,
		"mine":   mine,
	})
}

// UpsertPracticality upserts a user's practicality rating.
// PUT /api/toolset/:id/practicality
func (h *ToolsetHandler) UpsertPracticality(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Rate int `json:"rate" validate:"required,min=1,max=5"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var existing model.GalgameToolsetPracticality
	result := h.db.Where("toolset_id = ? AND user_id = ?", id, user.UID).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		h.db.Create(&model.GalgameToolsetPracticality{
			Rate:      req.Rate,
			UserID:    user.UID,
			ToolsetID: id,
		})
	} else {
		h.db.Model(&existing).Update("rate", req.Rate)
	}

	return response.OKMessage(c, "评分成功")
}

// ══════════════════════════════════════════
// Comments
// ══════════════════════════════════════════

// GetComments returns paginated comments for a toolset.
// GET /api/toolset/:id/comment
func (h *ToolsetHandler) GetComments(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=100"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	var total int64
	h.db.Model(&model.GalgameToolsetComment{}).Where("toolset_id = ?", id).Count(&total)

	var comments []model.GalgameToolsetComment
	offset := (req.Page - 1) * req.Limit
	h.db.Where("toolset_id = ?", id).
		Order("created DESC").
		Offset(offset).Limit(req.Limit).
		Find(&comments)

	type commentItem struct {
		model.GalgameToolsetComment
		User       userModel.UserBrief  `json:"user"`
		ParentUser *userModel.UserBrief `json:"parent_user,omitempty"`
	}

	items := make([]commentItem, 0, len(comments))
	for _, cm := range comments {
		var u userModel.UserBrief
		h.db.Where("id = ?", cm.UserID).First(&u)

		item := commentItem{
			GalgameToolsetComment: cm,
			User:                  u,
		}

		// If this is a reply, fetch parent comment's user
		if cm.ParentID != nil {
			var parent model.GalgameToolsetComment
			if err := h.db.First(&parent, *cm.ParentID).Error; err == nil {
				var pu userModel.UserBrief
				h.db.Where("id = ?", parent.UserID).First(&pu)
				item.ParentUser = &pu
			}
		}

		items = append(items, item)
	}

	return response.Paginated(c, items, total)
}

// CreateComment creates a comment on a toolset.
// POST /api/toolset/:id/comment
func (h *ToolsetHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Content  string `json:"content" validate:"required,min=1,max=1007"`
		ParentID *int   `json:"parentId"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	// Verify toolset exists
	var toolset model.GalgameToolset
	if err := h.db.First(&toolset, id).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该工具"))
	}

	comment := model.GalgameToolsetComment{
		Content:   req.Content,
		UserID:    user.UID,
		ToolsetID: id,
		ParentID:  req.ParentID,
	}
	if err := h.db.Create(&comment).Error; err != nil {
		return response.Error(c, errors.ErrInternal("发表评论失败"))
	}

	// Send notification to toolset owner or parent comment owner
	go func() {
		receiverID := toolset.UserID
		msgType := "commented"

		if req.ParentID != nil {
			var parent model.GalgameToolsetComment
			if err := h.db.First(&parent, *req.ParentID).Error; err == nil {
				receiverID = parent.UserID
				msgType = "replied"
			}
		}

		if receiverID != user.UID {
			h.db.Create(&msgModel.Message{
				Content:    markdown.ToPlainText(req.Content, 100),
				Link:       fmt.Sprintf("/toolset/%d", id),
				Type:       msgType,
				SenderID:   user.UID,
				ReceiverID: receiverID,
			})
		}
	}()

	return response.OK(c, comment)
}

// UpdateComment edits a comment (owner only).
// PUT /api/toolset/:id/comment
func (h *ToolsetHandler) UpdateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int    `json:"commentId" validate:"required,min=1"`
		Content   string `json:"content" validate:"required,min=1,max=1007"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var comment model.GalgameToolsetComment
	if err := h.db.First(&comment, req.CommentID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该评论"))
	}

	if comment.UserID != user.UID {
		return response.Error(c, errors.ErrForbidden("您只能编辑自己的评论"))
	}

	now := time.Now()
	h.db.Model(&comment).Updates(map[string]any{
		"content": req.Content,
		"edited":  now,
	})

	return response.OKMessage(c, "评论更新成功")
}

// DeleteComment deletes a comment.
// DELETE /api/toolset/:id/comment
func (h *ToolsetHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		CommentID int `query:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var comment model.GalgameToolsetComment
	if err := h.db.First(&comment, req.CommentID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该评论"))
	}

	// Owner of comment, toolset owner, or role >= 2
	var toolset model.GalgameToolset
	h.db.First(&toolset, id)

	if comment.UserID != user.UID && toolset.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此评论"))
	}

	h.db.Delete(&comment)
	return response.OKMessage(c, "评论已删除")
}

// ══════════════════════════════════════════
// Resources
// ══════════════════════════════════════════

// GetResourceDetail returns a resource and increments download count.
// GET /api/toolset/:id/resource/detail
func (h *ToolsetHandler) GetResourceDetail(c *fiber.Ctx) error {
	var req struct {
		ResourceID int `query:"resourceId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var resource model.GalgameToolsetResource
	if err := h.db.First(&resource, req.ResourceID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该资源"))
	}

	// Download +1
	go h.db.Model(&model.GalgameToolsetResource{}).Where("id = ?", resource.ID).
		Update("download", gorm.Expr("download + 1"))

	var user userModel.UserBrief
	h.db.Where("id = ?", resource.UserID).First(&user)

	return response.OK(c, fiber.Map{
		"resource": resource,
		"user":     user,
	})
}

// CreateResource creates a new resource for a toolset.
// POST /api/toolset/:id/resource
func (h *ToolsetHandler) CreateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Content  string `json:"content" validate:"max=1007"`
		Type     string `json:"type" validate:"required,oneof=s3 user"`
		Code     string `json:"code" validate:"max=1007"`
		Password string `json:"password" validate:"max=1007"`
		Size     string `json:"size" validate:"max=107"`
		Note     string `json:"note" validate:"max=1007"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	// Verify toolset exists
	var toolset model.GalgameToolset
	if err := h.db.First(&toolset, id).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该工具"))
	}

	var resource model.GalgameToolsetResource
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		resource = model.GalgameToolsetResource{
			Content:   req.Content,
			Type:      req.Type,
			Code:      req.Code,
			Password:  req.Password,
			Size:      req.Size,
			Note:      req.Note,
			ToolsetID: id,
			UserID:    user.UID,
		}
		if err := tx.Create(&resource).Error; err != nil {
			return err
		}

		// Moemoepoint +3
		tx.Model(&userModel.User{}).Where("id = ?", user.UID).
			Update("moemoepoint", gorm.Expr("moemoepoint + ?", 3))

		// Add contributor (ignore duplicate)
		var cnt int64
		tx.Model(&model.GalgameToolsetContributor{}).
			Where("toolset_id = ? AND user_id = ?", id, user.UID).Count(&cnt)
		if cnt == 0 {
			tx.Create(&model.GalgameToolsetContributor{
				ToolsetID: id,
				UserID:    user.UID,
			})
		}

		// Update resource_update_time
		tx.Model(&model.GalgameToolset{}).Where("id = ?", id).
			Update("resource_update_time", time.Now())

		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("创建资源失败"))
	}

	return response.OK(c, resource)
}

// UpdateResource updates a resource.
// PUT /api/toolset/:id/resource
func (h *ToolsetHandler) UpdateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ResourceID int    `json:"resourceId" validate:"required,min=1"`
		Content    string `json:"content" validate:"max=1007"`
		Code       string `json:"code" validate:"max=1007"`
		Password   string `json:"password" validate:"max=1007"`
		Size       string `json:"size" validate:"max=107"`
		Note       string `json:"note" validate:"max=1007"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var resource model.GalgameToolsetResource
	if err := h.db.First(&resource, req.ResourceID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该资源"))
	}

	if resource.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限编辑此资源"))
	}

	now := time.Now()
	updates := map[string]any{
		"password": req.Password,
		"note":     req.Note,
		"edited":   now,
	}

	// S3 type: only password and note can be updated
	// User type: all fields can be updated
	if resource.Type == "user" {
		updates["content"] = req.Content
		updates["code"] = req.Code
		updates["size"] = req.Size
	}

	h.db.Model(&resource).Updates(updates)
	return response.OKMessage(c, "资源更新成功")
}

// DeleteResource deletes a resource.
// DELETE /api/toolset/:id/resource
func (h *ToolsetHandler) DeleteResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ResourceID int `query:"resourceId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var resource model.GalgameToolsetResource
	if err := h.db.First(&resource, req.ResourceID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该资源"))
	}

	if resource.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此资源"))
	}

	// S3 cleanup
	if resource.Type == "s3" && resource.Code != "" {
		if err := h.s3.Delete(context.Background(), resource.Code); err != nil {
			slog.Warn("删除 S3 资源失败", "key", resource.Code, "error", err)
		}
	}

	h.db.Delete(&resource)

	// Moemoepoint -3
	h.db.Model(&userModel.User{}).Where("id = ?", resource.UserID).
		Update("moemoepoint", gorm.Expr("moemoepoint - ?", 3))

	return response.OKMessage(c, "资源已删除")
}

// ══════════════════════════════════════════
// Upload
// ══════════════════════════════════════════

const (
	maxSmallFileSize = 50 * 1024 * 1024         // 50MB
	maxLargeFileSize = 2 * 1024 * 1024 * 1024   // 2GB
	chunkSize        = 5 * 1024 * 1024           // 5MB
	uploadTTL        = 3600 * time.Second
	presignExpires   = 3600 * time.Second
)

var allowedArchiveExts = map[string]bool{
	".7z": true, ".zip": true, ".rar": true,
}

// uploadCacheEntry is stored in Redis during upload.
type uploadCacheEntry struct {
	Key      string `json:"key"`
	Type     string `json:"type"` // "small" or "multipart"
	Salt     string `json:"salt"`
	FileSize int64  `json:"filesize"`
	Base     string `json:"base"`
	Ext      string `json:"ext"`
	UploadID string `json:"upload_id,omitempty"` // multipart only
}

func generateSalt() string {
	b := make([]byte, 4) // 4 bytes → 8 hex chars, we take 7
	rand.Read(b)
	return hex.EncodeToString(b)[:7]
}

func buildS3Key(toolsetID, uid int, base, salt, ext string) string {
	return fmt.Sprintf("toolset/%d/%d_%s_%s%s", toolsetID, uid, base, salt, ext)
}

// UploadSmall generates a presigned PUT URL for files <= 50MB.
// POST /api/toolset/:id/upload/small
func (h *ToolsetHandler) UploadSmall(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Filename    string `json:"filename" validate:"required"`
		FileSize    int64  `json:"filesize" validate:"required,min=1"`
		ContentType string `json:"contentType" validate:"required"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if req.FileSize > maxSmallFileSize {
		return response.Error(c, errors.ErrBadRequest("小文件上传大小不能超过 50MB"))
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if !allowedArchiveExts[ext] {
		return response.Error(c, errors.ErrBadRequest("仅支持 .7z, .zip, .rar 格式"))
	}

	base := strings.TrimSuffix(req.Filename, ext)
	salt := generateSalt()
	key := buildS3Key(id, user.UID, base, salt, ext)

	presignedURL, err := h.s3.PresignPutObject(c.Context(), key, req.ContentType, presignExpires)
	if err != nil {
		return response.Error(c, errors.ErrInternal("生成上传链接失败"))
	}

	// Cache in Redis
	entry := uploadCacheEntry{
		Key:      key,
		Type:     "small",
		Salt:     salt,
		FileSize: req.FileSize,
		Base:     base,
		Ext:      ext,
	}
	data, _ := json.Marshal(entry)
	h.rdb.Set(c.Context(), "toolset:upload:"+salt, string(data), uploadTTL)

	return response.OK(c, fiber.Map{
		"presignedUrl": presignedURL,
		"salt":         salt,
		"key":          key,
	})
}

// UploadLarge initiates a multipart upload for files > 50MB <= 2GB.
// POST /api/toolset/:id/upload/large
func (h *ToolsetHandler) UploadLarge(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req struct {
		Filename    string `json:"filename" validate:"required"`
		FileSize    int64  `json:"filesize" validate:"required,min=1"`
		ContentType string `json:"contentType" validate:"required"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if req.FileSize > maxLargeFileSize {
		return response.Error(c, errors.ErrBadRequest("文件大小不能超过 2GB"))
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if !allowedArchiveExts[ext] {
		return response.Error(c, errors.ErrBadRequest("仅支持 .7z, .zip, .rar 格式"))
	}

	base := strings.TrimSuffix(req.Filename, ext)
	salt := generateSalt()
	key := buildS3Key(id, user.UID, base, salt, ext)

	uploadID, err := h.s3.CreateMultipartUpload(c.Context(), key, req.ContentType)
	if err != nil {
		return response.Error(c, errors.ErrInternal("创建分片上传失败"))
	}

	// Calculate number of parts
	numParts := int((req.FileSize + chunkSize - 1) / chunkSize)

	// Generate presigned URLs for each part
	partURLs := make([]fiber.Map, 0, numParts)
	for i := 1; i <= numParts; i++ {
		partURL, err := h.s3.PresignUploadPart(c.Context(), key, uploadID, int32(i), presignExpires)
		if err != nil {
			// Abort on failure
			h.s3.AbortMultipartUpload(c.Context(), key, uploadID)
			return response.Error(c, errors.ErrInternal("生成分片上传链接失败"))
		}
		partURLs = append(partURLs, fiber.Map{
			"partNumber":   i,
			"presignedUrl": partURL,
		})
	}

	// Cache in Redis
	entry := uploadCacheEntry{
		Key:      key,
		Type:     "multipart",
		Salt:     salt,
		FileSize: req.FileSize,
		Base:     base,
		Ext:      ext,
		UploadID: uploadID,
	}
	data, _ := json.Marshal(entry)
	h.rdb.Set(c.Context(), "toolset:upload:"+salt, string(data), uploadTTL)

	return response.OK(c, fiber.Map{
		"uploadId": uploadID,
		"salt":     salt,
		"key":      key,
		"parts":    partURLs,
	})
}

// UploadComplete completes a multipart upload and verifies size.
// POST /api/toolset/:id/upload/complete
func (h *ToolsetHandler) UploadComplete(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Salt  string `json:"salt" validate:"required"`
		Parts []struct {
			PartNumber int32  `json:"partNumber"`
			ETag       string `json:"etag"`
		} `json:"parts"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	// Read cache
	ctx := c.Context()
	val, err := h.rdb.Get(ctx, "toolset:upload:"+req.Salt).Result()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("上传会话不存在或已过期"))
	}

	var entry uploadCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return response.Error(c, errors.ErrInternal("解析上传缓存失败"))
	}

	// Complete multipart upload if applicable
	if entry.Type == "multipart" {
		if len(req.Parts) == 0 {
			return response.Error(c, errors.ErrBadRequest("分片信息不能为空"))
		}
		completedParts := make([]types.CompletedPart, 0, len(req.Parts))
		for _, p := range req.Parts {
			etag := p.ETag
			pn := p.PartNumber
			completedParts = append(completedParts, types.CompletedPart{
				ETag:       &etag,
				PartNumber: &pn,
			})
		}
		if err := h.s3.CompleteMultipartUpload(context.Background(), entry.Key, entry.UploadID, completedParts); err != nil {
			return response.Error(c, errors.ErrInternal("完成分片上传失败"))
		}
	}

	// Verify size via HeadObject
	actualSize, err := h.s3.HeadObject(context.Background(), entry.Key)
	if err != nil {
		slog.Warn("HeadObject 失败", "key", entry.Key, "error", err)
	} else if actualSize != entry.FileSize {
		slog.Warn("文件大小不匹配",
			"expected", entry.FileSize, "actual", actualSize, "key", entry.Key)
	}

	// Update daily upload count
	h.db.Model(&userModel.User{}).Where("id = ?", user.UID).
		Update("daily_toolset_upload_count", gorm.Expr("daily_toolset_upload_count + 1"))

	// Clean up Redis cache
	h.rdb.Del(ctx, "toolset:upload:"+req.Salt)

	return response.OK(c, fiber.Map{
		"key":  entry.Key,
		"size": actualSize,
	})
}

// UploadAbort aborts a multipart upload and cleans up cache.
// POST /api/toolset/:id/upload/abort
func (h *ToolsetHandler) UploadAbort(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Salt string `json:"salt" validate:"required"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	ctx := c.Context()
	val, err := h.rdb.Get(ctx, "toolset:upload:"+req.Salt).Result()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("上传会话不存在或已过期"))
	}

	var entry uploadCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return response.Error(c, errors.ErrInternal("解析上传缓存失败"))
	}

	// Abort multipart upload if applicable
	if entry.Type == "multipart" && entry.UploadID != "" {
		if err := h.s3.AbortMultipartUpload(context.Background(), entry.Key, entry.UploadID); err != nil {
			slog.Warn("中止分片上传失败", "key", entry.Key, "error", err)
		}
	}

	// For small uploads, try to delete the object if it was already uploaded
	if entry.Type == "small" {
		h.s3.Delete(context.Background(), entry.Key)
	}

	// Clean up cache
	h.rdb.Del(ctx, "toolset:upload:"+req.Salt)

	return response.OKMessage(c, "上传已取消")
}
