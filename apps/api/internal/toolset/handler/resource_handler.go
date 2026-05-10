package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ResourceHandler struct {
	resourceService *service.ResourceService
}

func NewResourceHandler(resourceService *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resourceService: resourceService}
}

// GetResourceDetail returns a resource and increments download count.
// GET /api/toolset/:id/resource/detail
func (h *ResourceHandler) GetResourceDetail(c *fiber.Ctx) error {
	var req dto.ResourceDetailRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	detail, appErr := h.resourceService.GetResourceDetail(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// CreateResource creates a new resource for a toolset.
// POST /api/toolset/:id/resource
func (h *ResourceHandler) CreateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.CreateResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resource, appErr := h.resourceService.CreateResource(user.UID, id, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, resource)
}

// UpdateResource updates a resource.
// PUT /api/toolset/:id/resource
func (h *ResourceHandler) UpdateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.resourceService.UpdateResource(user.UID, user.Role, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源更新成功")
}

// DeleteResource deletes a resource.
// DELETE /api/toolset/:id/resource
func (h *ResourceHandler) DeleteResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteResourceRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.resourceService.DeleteResource(user.UID, user.Role, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已删除")
}
