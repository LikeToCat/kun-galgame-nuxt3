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

// GalgameHandler groups the "core" galgame endpoints: create, merge PR,
// detail aggregation, list, and local interactions.
type GalgameHandler struct {
	galgameService *service.GalgameService
}

func NewGalgameHandler(galgameService *service.GalgameService) *GalgameHandler {
	return &GalgameHandler{galgameService: galgameService}
}

// ──────────────────────────────────────────
// Create / MergePR (proxy to wiki with local side effects)
// ──────────────────────────────────────────

// Create — POST /api/galgame
func (h *GalgameHandler) Create(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := c.Get("X-OAuth-Token")
	if token == "" {
		return response.Error(c, errors.ErrBadRequest("缺少 OAuth 访问令牌"))
	}

	data, appErr := h.galgameService.Create(c.Context(), user.UID, token, c.Body())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// MergePR — PUT /api/galgame/:gid/prs/:id/merge
func (h *GalgameHandler) MergePR(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := c.Get("X-OAuth-Token")
	data, appErr := h.galgameService.MergePR(
		c.Context(), user.UID, c.Params("gid"), c.Params("id"), token,
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// ──────────────────────────────────────────
// GetDetail / GetList
// ──────────────────────────────────────────

// GetDetail — GET /api/galgame/:gid
func (h *GalgameHandler) GetDetail(c *fiber.Ctx) error {
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	detail, appErr := h.galgameService.GetDetail(c.Context(), gid, optionalUID(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// GetList — GET /api/galgame
func (h *GalgameHandler) GetList(c *fiber.Ctx) error {
	var req dto.GalgameListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.galgameService.GetList(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// ──────────────────────────────────────────
// Interactions
// ──────────────────────────────────────────

// ToggleLike — PUT /api/galgame/:gid/like
func (h *GalgameHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleLike(c.Context(), user.UID, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite — PUT /api/galgame/:gid/favorite
func (h *GalgameHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleFavorite(user.UID, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}
