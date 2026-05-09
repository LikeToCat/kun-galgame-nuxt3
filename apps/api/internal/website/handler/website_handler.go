package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// WebsiteHandler handles the core website CRUD + interaction routes.
// Sub-domains (comments, category, tag) each live in their own handler file.
type WebsiteHandler struct {
	websiteService *service.WebsiteService
}

func NewWebsiteHandler(websiteService *service.WebsiteService) *WebsiteHandler {
	return &WebsiteHandler{websiteService: websiteService}
}

// GetWebsites returns all websites as WebsiteCard[].
// GET /api/website
func (h *WebsiteHandler) GetWebsites(c *fiber.Ctx) error {
	return response.OK(c, h.websiteService.GetList())
}

// CreateWebsite creates a new website entry.
// POST /api/website
func (h *WebsiteHandler) CreateWebsite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateWebsiteRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.websiteService.Create(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "网站创建成功")
}

// GetWebsiteDetail returns website detail by domain.
// GET /api/website/:domain
func (h *WebsiteHandler) GetWebsiteDetail(c *fiber.Ctx) error {
	domain := c.Params("domain")

	currentUserID := 0
	if u := middleware.GetUser(c); u != nil {
		currentUserID = u.UID
	}

	detail, appErr := h.websiteService.GetDetail(c.Context(), domain, currentUserID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// UpdateWebsite updates a website.
// PUT /api/website/:domain
func (h *WebsiteHandler) UpdateWebsite(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateWebsiteRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.websiteService.Update(&req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "网站更新成功")
}

// DeleteWebsite deletes a website.
// DELETE /api/website/:domain
func (h *WebsiteHandler) DeleteWebsite(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteWebsiteRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.websiteService.Delete(req.WebsiteID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "网站已删除")
}

// ToggleLike toggles website like.
// PUT /api/website/:domain/like
func (h *WebsiteHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.ToggleInteractionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.websiteService.ToggleLike(user.UID, req.WebsiteID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite toggles website favorite.
// PUT /api/website/:domain/favorite
func (h *WebsiteHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.ToggleInteractionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.websiteService.ToggleFavorite(user.UID, req.WebsiteID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}
