package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
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

// GetResourceList returns the latest galgame resources.
// GET /api/galgame-resource
func (h *ResourceHandler) GetResourceList(c *fiber.Ctx) error {
	var req dto.ResourceListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.resourceService.GetResourceList(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetResourceDetail returns a single resource with galgame info and recommendations.
// GET /api/galgame-resource/:id
func (h *ResourceHandler) GetResourceDetail(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的资源 ID"))
	}

	currentUID := optionalUID(c)
	detail, notFound, appErr := h.resourceService.GetResourceDetail(c.Context(), id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if notFound != nil {
		// Legacy "not found" string response expected by the frontend.
		return response.OK(c, "not found")
	}
	return response.OK(c, detail)
}

// GetResourceDownloadDetail returns resource detail with download links.
// GET /api/galgame-resource/:id/detail
func (h *ResourceHandler) GetResourceDownloadDetail(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的资源 ID"))
	}

	currentUID := optionalUID(c)
	detail, appErr := h.resourceService.GetResourceDownloadDetail(id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// GetGalgameResources returns resources for a specific galgame.
// GET /api/galgame/:gid/resource/all
func (h *ResourceHandler) GetGalgameResources(c *fiber.Ctx) error {
	var req dto.GalgameResourcesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	cards, appErr := h.resourceService.GetGalgameResources(&req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, cards)
}

// optionalUID returns the logged-in user's ID from OptionalAuth middleware,
// or 0 if not authenticated.
func optionalUID(c *fiber.Ctx) int {
	if user := middleware.GetUser(c); user != nil {
		return user.UID
	}
	return 0
}
