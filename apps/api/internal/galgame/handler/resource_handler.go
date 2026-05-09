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

// resourceIDFromQueryOrBody pulls galgameResourceId from either body or query
// — useful for status PUT endpoints that the frontend hits with body params.

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
	detail, appErr := h.resourceService.GetResourceDownloadDetail(c.Context(), id, currentUID)
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

	cards, appErr := h.resourceService.GetGalgameResources(c.Context(), &req)
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

// CreateResource — POST /api/galgame/:gid/resource
func (h *ResourceHandler) CreateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateGalgameResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.CreateResource(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源创建成功")
}

// UpdateResource — PUT /api/galgame/:gid/resource
func (h *ResourceHandler) UpdateResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateGalgameResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.UpdateResource(user.UID, user.Role, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源更新成功")
}

// DeleteResource — DELETE /api/galgame/:gid/resource
func (h *ResourceHandler) DeleteResource(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteGalgameResourceRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.DeleteResource(user.UID, user.Role, req.GalgameResourceID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已删除")
}

// ToggleLike — PUT /api/galgame/:gid/resource/like
func (h *ResourceHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ToggleResourceLikeRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.ToggleLike(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// MarkValid — PUT /api/galgame/:gid/resource/valid
func (h *ResourceHandler) MarkValid(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ResourceStatusRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.MarkValid(user.UID, req.GalgameResourceID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已标记为有效")
}

// MarkExpired — PUT /api/galgame/:gid/resource/expired
func (h *ResourceHandler) MarkExpired(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ResourceStatusRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.MarkExpired(user.UID, req.GalgameResourceID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已标记为失效")
}

// GetRecommend — GET /api/galgame-resource/:id/recommend
// Returns the top 6 sibling resources (same galgame, sorted by like_count).
func (h *ResourceHandler) GetRecommend(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的资源 ID"))
	}
	cards, appErr := h.resourceService.GetRecommendations(c.Context(), id)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, cards)
}
