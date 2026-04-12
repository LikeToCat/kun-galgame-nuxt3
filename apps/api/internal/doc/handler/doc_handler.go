package handler

import (
	"time"

	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DocHandler struct {
	db *gorm.DB
}

func NewDocHandler(db *gorm.DB) *DocHandler {
	return &DocHandler{db: db}
}

// ── Article ─────────────────────────────

// GetArticles returns paginated article list.
// GET /api/doc/article
func (h *DocHandler) GetArticles(c *fiber.Ctx) error {
	var req struct {
		Page       int    `query:"page" validate:"min=1"`
		Limit      int    `query:"limit" validate:"min=1,max=50"`
		CategoryID *int   `query:"categoryId"`
		TagID      *int   `query:"tagId"`
		Status     *int   `query:"status"`
		IsPin      *bool  `query:"isPin"`
		Keyword    string `query:"keyword"`
		OrderBy    string `query:"orderBy" validate:"omitempty,oneof=published_time created view updated"`
		SortOrder  string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.OrderBy == "" {
		req.OrderBy = "published_time"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	var articles []model.DocArticle
	var total int64

	query := h.db.Model(&model.DocArticle{})
	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.IsPin != nil {
		query = query.Where("is_pin = ?", *req.IsPin)
	}
	if req.Keyword != "" {
		query = query.Where("title ILIKE ? OR slug ILIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	if req.TagID != nil {
		query = query.Where("id IN (SELECT doc_article_id FROM doc_article_tag_relation WHERE doc_tag_id = ?)", *req.TagID)
	}

	query.Count(&total)
	query.Order(req.OrderBy + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&articles)

	return response.Paginated(c, articles, total)
}

// GetArticleBySlug returns a single article by slug.
// GET /api/doc/article/:slug
func (h *DocHandler) GetArticleBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var article model.DocArticle
	if err := h.db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该文章"))
	}

	go h.db.Model(&model.DocArticle{}).Where("id = ?", article.ID).
		Update("view", gorm.Expr("view + 1"))

	return response.OK(c, article)
}

// CreateArticle creates a new doc article.
// POST /api/doc/article
func (h *DocHandler) CreateArticle(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Title           string `json:"title" validate:"required,max=233"`
		Slug            string `json:"slug" validate:"required,max=233"`
		Description     string `json:"description" validate:"max=1000"`
		Banner          string `json:"banner" validate:"max=500"`
		Status          int    `json:"status" validate:"oneof=0 1 2"`
		IsPin           bool   `json:"isPin"`
		ContentMarkdown string `json:"contentMarkdown" validate:"required"`
		CategoryID      int    `json:"categoryId" validate:"required,min=1"`
		TagIDs          []int  `json:"tagIds"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	now := time.Now()
	article := model.DocArticle{
		Title: req.Title, Slug: req.Slug, Path: "/doc/" + req.Slug,
		Description: req.Description, Banner: req.Banner,
		Status: req.Status, IsPin: req.IsPin,
		ContentMarkdown: req.ContentMarkdown,
		CategoryID: req.CategoryID, AuthorID: user.UID,
		PublishedTime: now,
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&article).Error; err != nil {
			return err
		}
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.DocArticleTagRelation{
				DocArticleID: article.ID, DocTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("创建文章失败"))
	}

	return response.OK(c, article)
}

// UpdateArticle updates an existing article.
// PUT /api/doc/article
func (h *DocHandler) UpdateArticle(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ArticleID       int    `json:"articleId" validate:"required,min=1"`
		Title           string `json:"title" validate:"required,max=233"`
		Slug            string `json:"slug" validate:"required,max=233"`
		Description     string `json:"description" validate:"max=1000"`
		Banner          string `json:"banner" validate:"max=500"`
		Status          int    `json:"status" validate:"oneof=0 1 2"`
		IsPin           bool   `json:"isPin"`
		ContentMarkdown string `json:"contentMarkdown" validate:"required"`
		CategoryID      int    `json:"categoryId" validate:"required,min=1"`
		TagIDs          []int  `json:"tagIds"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	now := time.Now()
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.DocArticle{}).Where("id = ?", req.ArticleID).Updates(map[string]any{
			"title": req.Title, "slug": req.Slug, "path": "/doc/" + req.Slug,
			"description": req.Description, "banner": req.Banner,
			"status": req.Status, "is_pin": req.IsPin,
			"content_markdown": req.ContentMarkdown,
			"category_id": req.CategoryID, "edited_time": &now,
		}).Error; err != nil {
			return err
		}
		tx.Where("doc_article_id = ?", req.ArticleID).Delete(&model.DocArticleTagRelation{})
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.DocArticleTagRelation{
				DocArticleID: req.ArticleID, DocTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("更新文章失败"))
	}

	return response.OKMessage(c, "文章更新成功")
}

// DeleteArticle deletes a doc article.
// DELETE /api/doc/article
func (h *DocHandler) DeleteArticle(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ArticleID int `query:"articleId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Where("doc_article_id = ?", req.ArticleID).Delete(&model.DocArticleTagRelation{})
	h.db.Delete(&model.DocArticle{}, req.ArticleID)

	return response.OKMessage(c, "文章已删除")
}

// ── Category ────────────────────────────

// GetCategories returns doc category list.
// GET /api/doc/category
func (h *DocHandler) GetCategories(c *fiber.Ctx) error {
	var req struct {
		Page    int    `query:"page" validate:"min=1"`
		Limit   int    `query:"limit" validate:"min=1,max=50"`
		Keyword string `query:"keyword"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var categories []model.DocCategory
	var total int64
	query := h.db.Model(&model.DocCategory{})
	if req.Keyword != "" {
		query = query.Where("title ILIKE ? OR slug ILIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	query.Count(&total)
	query.Order("sort_order ASC, id ASC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&categories)

	return response.Paginated(c, categories, total)
}

// CreateCategory creates a doc category.
// POST /api/doc/category
func (h *DocHandler) CreateCategory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.DocCategory
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.db.Create(&req).Error; err != nil {
		return response.Error(c, errors.ErrInternal("创建分类失败"))
	}
	return response.OK(c, req)
}

// UpdateCategory updates a doc category.
// PUT /api/doc/category
func (h *DocHandler) UpdateCategory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		Slug        string `json:"slug" validate:"required"`
		Title       string `json:"title" validate:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		SortOrder   int    `json:"sortOrder"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Model(&model.DocCategory{}).Where("id = ?", req.CategoryID).Updates(map[string]any{
		"slug": req.Slug, "title": req.Title, "description": req.Description,
		"icon": req.Icon, "sort_order": req.SortOrder,
	})
	return response.OKMessage(c, "分类更新成功")
}

// DeleteCategory deletes a doc category.
// DELETE /api/doc/category
func (h *DocHandler) DeleteCategory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CategoryID int `query:"categoryId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Delete(&model.DocCategory{}, req.CategoryID)
	return response.OKMessage(c, "分类已删除")
}

// ── Tag ─────────────────────────────────

// GetTags returns doc tag list.
// GET /api/doc/tag
func (h *DocHandler) GetTags(c *fiber.Ctx) error {
	var req struct {
		Page    int    `query:"page" validate:"min=1"`
		Limit   int    `query:"limit" validate:"min=1,max=50"`
		Keyword string `query:"keyword"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var tags []model.DocTag
	var total int64
	query := h.db.Model(&model.DocTag{})
	if req.Keyword != "" {
		query = query.Where("title ILIKE ? OR slug ILIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	query.Count(&total)
	query.Order("title ASC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&tags)

	return response.Paginated(c, tags, total)
}

// CreateTag creates a doc tag.
// POST /api/doc/tag
func (h *DocHandler) CreateTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.DocTag
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.db.Create(&req).Error; err != nil {
		return response.Error(c, errors.ErrInternal("创建标签失败"))
	}
	return response.OK(c, req)
}

// DeleteTag deletes a doc tag.
// DELETE /api/doc/tag
func (h *DocHandler) DeleteTag(c *fiber.Ctx) error {
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
	h.db.Where("doc_tag_id = ?", req.TagID).Delete(&model.DocArticleTagRelation{})
	h.db.Delete(&model.DocTag{}, req.TagID)
	return response.OKMessage(c, "标签已删除")
}
