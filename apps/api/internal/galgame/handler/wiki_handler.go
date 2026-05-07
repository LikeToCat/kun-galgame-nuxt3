package handler

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// WikiHandler groups wiki pass-through endpoints and the galgame sub-routes
// that proxy to wiki + enrich with local user data.
type WikiHandler struct {
	wikiService *service.WikiService
}

func NewWikiHandler(wikiService *service.WikiService) *WikiHandler {
	return &WikiHandler{wikiService: wikiService}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// ProxyGet forwards a GET request to wiki service.
func (h *WikiHandler) ProxyGet(c *fiber.Ctx) error {
	data, appErr := h.wikiService.ProxyGet(c.Context(), c.Path(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// ProxyWriteWithToken returns a Fiber handler that forwards a POST/PUT/DELETE
// to wiki with the user's OAuth token.
func (h *WikiHandler) ProxyWriteWithToken(method string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, appErr := middleware.MustGetUser(c); appErr != nil {
			return response.Error(c, appErr)
		}

		token := c.Get("X-OAuth-Token")
		if token == "" {
			return response.Error(c, errors.ErrBadRequest("缺少 OAuth 访问令牌"))
		}

		data, appErr := h.wikiService.ProxyWrite(
			c.Context(), method, c.Path(), token, c.Body(), c.Get("Content-Type"),
		)
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
	}
}

// ──────────────────────────────────────────
// Galgame sub-routes
// ──────────────────────────────────────────

// GetGalgameLinks — GET /galgame/:gid/link/all
func (h *WikiHandler) GetGalgameLinks(c *fiber.Ctx) error {
	links, appErr := h.wikiService.GetGalgameLinks(c.Context(), c.Params("gid"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, links)
}

// GetGalgameHistory — GET /galgame/:gid/history/all
func (h *WikiHandler) GetGalgameHistory(c *fiber.Ctx) error {
	page, appErr := h.wikiService.GetGalgameHistory(
		c.Context(), c.Params("gid"), collectQuery(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, page.Items, page.Total)
}

// GetGalgamePRs — GET /galgame/:gid/pr/all
func (h *WikiHandler) GetGalgamePRs(c *fiber.Ctx) error {
	page, appErr := h.wikiService.GetGalgamePRs(
		c.Context(), c.Params("gid"), collectQuery(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, page.Items, page.Total)
}
