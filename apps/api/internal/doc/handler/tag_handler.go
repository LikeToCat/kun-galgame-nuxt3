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

type TagHandler struct {
	tagService *service.TagService
}

func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

// GetTags returns doc tag list.
// GET /api/doc/tag
func (h *TagHandler) GetTags(c *fiber.Ctx) error {
	var req dto.GetTagsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result := h.tagService.GetList(&req)
	return response.Paginated(c, result.Items, result.Total)
}

// CreateTag creates a doc tag.
// POST /api/doc/tag
func (h *TagHandler) CreateTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.DocTag
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.tagService.Create(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, req)
}

// UpdateTag updates an existing doc tag.
// PUT /api/doc/tag
func (h *TagHandler) UpdateTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateTagRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	tag, appErr := h.tagService.Update(&req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, tag)
}

// DeleteTag deletes a doc tag.
// DELETE /api/doc/tag
func (h *TagHandler) DeleteTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteTagRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.tagService.Delete(req.TagID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "标签已删除")
}
