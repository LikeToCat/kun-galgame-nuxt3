package handler

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ArticleHandler struct {
	articleService *service.ArticleService
}

func NewArticleHandler(articleService *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// GetArticles returns paginated article list.
// GET /api/doc/article
func (h *ArticleHandler) GetArticles(c *fiber.Ctx) error {
	var req dto.GetArticlesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result := h.articleService.GetList(&req)
	return response.Paginated(c, result.Items, result.Total)
}

// GetArticleBySlug returns a single article by slug.
// GET /api/doc/article/:slug
func (h *ArticleHandler) GetArticleBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	detail, appErr := h.articleService.GetBySlug(slug)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// CreateArticle creates a new doc article.
// POST /api/doc/article
func (h *ArticleHandler) CreateArticle(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateArticleRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	article, appErr := h.articleService.Create(user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, article)
}

// UpdateArticle updates an existing article.
// PUT /api/doc/article
func (h *ArticleHandler) UpdateArticle(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateArticleRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.articleService.Update(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "文章更新成功")
}

// DeleteArticle deletes a doc article.
// DELETE /api/doc/article
func (h *ArticleHandler) DeleteArticle(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteArticleRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.articleService.Delete(req.ArticleID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "文章已删除")
}
