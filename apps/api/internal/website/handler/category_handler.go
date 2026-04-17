package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// GetWebsiteCategory returns a category with its websites.
// GET /api/website-category/:name
func (h *CategoryHandler) GetWebsiteCategory(c *fiber.Ctx) error {
	name := c.Params("name")
	detail, appErr := h.categoryService.GetDetail(name)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// UpdateWebsiteCategory updates a website category.
// PUT /api/website-category
func (h *CategoryHandler) UpdateWebsiteCategory(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateWebsiteCategoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.categoryService.Update(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "分类更新成功")
}
