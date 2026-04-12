package handler

import (
	"net/url"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebsiteHandler struct {
	db *gorm.DB
}

func NewWebsiteHandler(db *gorm.DB) *WebsiteHandler {
	return &WebsiteHandler{db: db}
}

// ── Website CRUD ────────────────────────

// GetWebsites returns all websites.
// GET /api/website
func (h *WebsiteHandler) GetWebsites(c *fiber.Ctx) error {
	var websites []model.GalgameWebsite
	h.db.Order("created DESC").Find(&websites)
	return response.OK(c, websites)
}

// CreateWebsite creates a new website entry.
// POST /api/website
func (h *WebsiteHandler) CreateWebsite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Name        string `json:"name" validate:"required,max=233"`
		URL         string `json:"url" validate:"required,url,max=500"`
		Description string `json:"description" validate:"max=1000"`
		Icon        string `json:"icon" validate:"max=500"`
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
		Language    string `json:"language" validate:"max=10"`
		TagIDs      []int  `json:"tag_ids"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	parsedURL, _ := url.Parse(req.URL)
	domain := ""
	if parsedURL != nil {
		domain = parsedURL.Hostname()
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		website := model.GalgameWebsite{
			Name: req.Name, URL: req.URL, Description: req.Description,
			Icon: req.Icon, Language: req.Language, AgeLimit: req.AgeLimit,
			CategoryID: req.CategoryID, UserID: user.UID,
		}
		_ = domain // domain stored in jsonb field if needed
		if err := tx.Create(&website).Error; err != nil {
			return err
		}
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
				GalgameWebsiteID: website.ID, GalgameWebsiteTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("创建网站失败"))
	}

	return response.OKMessage(c, "网站创建成功")
}

// GetWebsiteDetail returns website detail by domain.
// GET /api/website/:domain
func (h *WebsiteHandler) GetWebsiteDetail(c *fiber.Ctx) error {
	domain := c.Params("domain")

	var website model.GalgameWebsite
	if err := h.db.Where("url ILIKE ?", "%"+domain+"%").First(&website).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该网站"))
	}

	go h.db.Model(&model.GalgameWebsite{}).Where("id = ?", website.ID).
		Update("view", gorm.Expr("view + 1"))

	// Fetch interactions
	userInfo := middleware.GetUser(c)
	isLiked := false
	isFavorited := false
	if userInfo != nil {
		var likeCount, favCount int64
		h.db.Model(&model.GalgameWebsiteLike{}).Where("user_id = ? AND website_id = ?", userInfo.UID, website.ID).Count(&likeCount)
		h.db.Model(&model.GalgameWebsiteFavorite{}).Where("user_id = ? AND website_id = ?", userInfo.UID, website.ID).Count(&favCount)
		isLiked = likeCount > 0
		isFavorited = favCount > 0
	}

	return response.OK(c, fiber.Map{
		"website":     website,
		"isLiked":     isLiked,
		"isFavorited": isFavorited,
	})
}

// UpdateWebsite updates a website.
// PUT /api/website/:domain
func (h *WebsiteHandler) UpdateWebsite(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID   int    `json:"websiteId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required,max=233"`
		URL         string `json:"url" validate:"required,url,max=500"`
		Description string `json:"description" validate:"max=1000"`
		Icon        string `json:"icon" validate:"max=500"`
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
		Language    string `json:"language" validate:"max=10"`
		TagIDs      []int  `json:"tag_ids"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).Updates(map[string]any{
			"name": req.Name, "url": req.URL, "description": req.Description,
			"icon": req.Icon, "category_id": req.CategoryID,
			"age_limit": req.AgeLimit, "language": req.Language,
		})
		tx.Where("website_id = ?", req.WebsiteID).Delete(&model.GalgameWebsiteTagRelation{})
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
				GalgameWebsiteID: req.WebsiteID, GalgameWebsiteTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("更新网站失败"))
	}

	return response.OKMessage(c, "网站更新成功")
}

// DeleteWebsite deletes a website.
// DELETE /api/website/:domain
func (h *WebsiteHandler) DeleteWebsite(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `query:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Delete(&model.GalgameWebsite{}, req.WebsiteID)
	return response.OKMessage(c, "网站已删除")
}

// ── Interactions ────────────────────────

// ToggleLike toggles website like.
// PUT /api/website/:domain/like
func (h *WebsiteHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `json:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameWebsiteLike
		result := tx.Where("user_id = ? AND website_id = ?", user.UID, req.WebsiteID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameWebsiteLike{UserID: user.UID, WebsiteID: req.WebsiteID})
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("like_count", gorm.Expr("like_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("like_count", gorm.Expr("like_count - 1"))
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite toggles website favorite.
// PUT /api/website/:domain/favorite
func (h *WebsiteHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `json:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameWebsiteFavorite
		result := tx.Where("user_id = ? AND website_id = ?", user.UID, req.WebsiteID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameWebsiteFavorite{UserID: user.UID, WebsiteID: req.WebsiteID})
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("favorite_count", gorm.Expr("favorite_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("favorite_count", gorm.Expr("favorite_count - 1"))
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// CreateComment creates a website comment.
// POST /api/website/:domain/comment
func (h *WebsiteHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Content   string `json:"content" validate:"required,min=1,max=1007"`
		WebsiteID int    `json:"websiteId" validate:"required,min=1"`
		ParentID  *int   `json:"parentId"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	comment := model.GalgameWebsiteComment{
		Content: req.Content, WebsiteID: req.WebsiteID,
		UserID: user.UID, ParentID: req.ParentID,
	}
	if err := h.db.Create(&comment).Error; err != nil {
		return response.Error(c, errors.ErrInternal("发表评论失败"))
	}

	h.db.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
		Update("comment_count", gorm.Expr("comment_count + 1"))

	return response.OK(c, comment)
}

// DeleteComment deletes a website comment.
// DELETE /api/website/:domain/comment
func (h *WebsiteHandler) DeleteComment(c *fiber.Ctx) error {
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

	var comment model.GalgameWebsiteComment
	if err := h.db.First(&comment, req.CommentID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该评论"))
	}
	if comment.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此评论"))
	}

	h.db.Delete(&comment)
	h.db.Model(&model.GalgameWebsite{}).Where("id = ?", comment.WebsiteID).
		Update("comment_count", gorm.Expr("comment_count - 1"))

	return response.OKMessage(c, "评论已删除")
}

// ── Category & Tag ──────────────────────

// GetWebsiteCategory returns a category with its websites.
// GET /api/website-category/:name
func (h *WebsiteHandler) GetWebsiteCategory(c *fiber.Ctx) error {
	name := c.Params("name")
	var category model.GalgameWebsiteCategory
	if err := h.db.Where("name = ?", name).First(&category).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该分类"))
	}

	var websites []model.GalgameWebsite
	h.db.Where("category_id = ?", category.ID).Find(&websites)

	return response.OK(c, fiber.Map{
		"category":     category,
		"websites":     websites,
		"websiteCount": len(websites),
	})
}

// UpdateWebsiteCategory updates a website category.
// PUT /api/website-category
func (h *WebsiteHandler) UpdateWebsiteCategory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required"`
		Label       string `json:"label"`
		Description string `json:"description"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Model(&model.GalgameWebsiteCategory{}).Where("id = ?", req.CategoryID).Updates(map[string]any{
		"name": req.Name, "label": req.Label, "description": req.Description,
	})
	return response.OKMessage(c, "分类更新成功")
}

// GetWebsiteTags returns website tags.
// GET /api/website-tag
func (h *WebsiteHandler) GetWebsiteTags(c *fiber.Ctx) error {
	var tags []model.GalgameWebsiteTag
	h.db.Order("id ASC").Find(&tags)
	return response.OK(c, tags)
}

// CreateWebsiteTag creates a website tag.
// POST /api/website-tag
func (h *WebsiteHandler) CreateWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.GalgameWebsiteTag
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Create(&req)
	return response.OKMessage(c, "标签创建成功")
}

// UpdateWebsiteTag updates a website tag.
// PUT /api/website-tag
func (h *WebsiteHandler) UpdateWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		TagID       int    `json:"tagId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required"`
		Label       string `json:"label"`
		Description string `json:"description"`
		Level       int    `json:"level"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Model(&model.GalgameWebsiteTag{}).Where("id = ?", req.TagID).Updates(map[string]any{
		"name": req.Name, "label": req.Label, "description": req.Description, "level": req.Level,
	})
	return response.OKMessage(c, "标签更新成功")
}

// DeleteWebsiteTag deletes a website tag.
// DELETE /api/website-tag
func (h *WebsiteHandler) DeleteWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		TagID int `query:"tagId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Where("tag_id = ?", req.TagID).Delete(&model.GalgameWebsiteTagRelation{})
	h.db.Delete(&model.GalgameWebsiteTag{}, req.TagID)
	return response.OKMessage(c, "标签已删除")
}
