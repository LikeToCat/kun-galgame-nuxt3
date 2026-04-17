package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/service"
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

// GetWebsiteTags returns all website tags.
// GET /api/website-tag
func (h *TagHandler) GetWebsiteTags(c *fiber.Ctx) error {
	return response.OK(c, h.tagService.GetAll())
}

// GetWebsiteTagDetail returns a tag with its websites.
// GET /api/website-tag/:name
func (h *TagHandler) GetWebsiteTagDetail(c *fiber.Ctx) error {
	name := c.Params("name")
	detail, appErr := h.tagService.GetDetail(name)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// CreateWebsiteTag creates a website tag.
// POST /api/website-tag
func (h *TagHandler) CreateWebsiteTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var tag model.GalgameWebsiteTag
	if appErr := utils.ParseAndValidate(c, &tag); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.tagService.Create(&tag); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "标签创建成功")
}

// UpdateWebsiteTag updates a website tag.
// PUT /api/website-tag
func (h *TagHandler) UpdateWebsiteTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateWebsiteTagRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.tagService.Update(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "标签更新成功")
}

// DeleteWebsiteTag deletes a website tag.
// DELETE /api/website-tag
func (h *TagHandler) DeleteWebsiteTag(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteWebsiteTagRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.tagService.Delete(req.TagID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "标签已删除")
}
