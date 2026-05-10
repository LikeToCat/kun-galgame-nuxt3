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

// ToolsetHandler handles the core toolset CRUD routes.
// Sub-domains (comments, practicality, resources, uploads) each live in their
// own handler file to keep each small and focused.
type ToolsetHandler struct {
	toolsetService *service.ToolsetService
}

func NewToolsetHandler(toolsetService *service.ToolsetService) *ToolsetHandler {
	return &ToolsetHandler{toolsetService: toolsetService}
}

// GetList returns a paginated list of toolsets with filters.
// GET /api/toolset
func (h *ToolsetHandler) GetList(c *fiber.Ctx) error {
	var req dto.ToolsetListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 24
	}

	items, total := h.toolsetService.GetList(c.Context(), &req)
	return response.Paginated(c, items, total)
}

// Create creates a new toolset.
// POST /api/toolset
func (h *ToolsetHandler) Create(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateToolsetRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	toolset, appErr := h.toolsetService.Create(user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, toolset)
}

// GetDetail returns toolset detail.
// GET /api/toolset/:id
func (h *ToolsetHandler) GetDetail(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	detail, appErr := h.toolsetService.GetDetail(c.Context(), id)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// Update updates a toolset.
// PUT /api/toolset/:id
func (h *ToolsetHandler) Update(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.UpdateToolsetRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.toolsetService.Update(user.UID, user.Role, id, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "工具更新成功")
}

// Delete deletes a toolset.
// DELETE /api/toolset/:id
func (h *ToolsetHandler) Delete(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	if appErr := h.toolsetService.Delete(user.UID, user.Role, id); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "工具已删除")
}
