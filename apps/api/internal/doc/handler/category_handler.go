package handler

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/service"
	"kun-galgame-api/internal/middleware"
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

// GetCategories returns doc category list.
// GET /api/doc/category
func (h *CategoryHandler) GetCategories(c *fiber.Ctx) error {
	var req dto.GetCategoriesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result := h.categoryService.GetList(&req)
	return response.Paginated(c, result.Items, result.Total)
}

// CreateCategory creates a doc category.
// POST /api/doc/category
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.DocCategory
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.categoryService.Create(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, req)
}

// UpdateCategory updates a doc category.
// PUT /api/doc/category
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateCategoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.categoryService.Update(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "分类更新成功")
}

// DeleteCategory deletes a doc category.
// DELETE /api/doc/category
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteCategoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.categoryService.Delete(req.CategoryID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "分类已删除")
}
