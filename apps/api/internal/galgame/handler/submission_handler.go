package handler

import (
	"encoding/json"
	"strconv"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// SubmissionHandler exposes the user submission endpoints (submit / claim /
// patch-draft / delete-draft / list-mine / search-with-pending). All of
// them require authentication; the OAuth access token is read from the
// session via middleware.GetAccessToken and forwarded to wiki so the wiki
// sees the authenticated kungal user, not whatever Authorization header
// the client tried to send.
type SubmissionHandler struct {
	svc *service.SubmissionService
}

func NewSubmissionHandler(svc *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc: svc}
}

// Submit — POST /api/galgame/submit
func (h *SubmissionHandler) Submit(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.Submit(c.Context(), token, c.Body(), c.Get("Content-Type"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// Claim — POST /api/galgame/:gid/claim
func (h *SubmissionHandler) Claim(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.Claim(c.Context(), user.UID, token, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// PatchDraft — PATCH /api/galgame/:gid (only valid for status IN (3,4) /
// own row; wiki enforces both)
func (h *SubmissionHandler) PatchDraft(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.PatchDraft(c.Context(), token, gid, c.Body(), c.Get("Content-Type"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// DeleteDraft — DELETE /api/galgame/:gid (only valid for status IN (3,4) /
// own row; wiki enforces both)
func (h *SubmissionHandler) DeleteDraft(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	if appErr := h.svc.DeleteDraft(c.Context(), token, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "撤回成功")
}

// ListMine — GET /api/galgame/mine
func (h *SubmissionHandler) ListMine(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.ListMine(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// SearchWithPending — GET /api/galgame/search/wizard
//
// Dedicated endpoint for the 发布向导 flow. Forces include_pending=true so
// callers don't accidentally use the public search and miss the "你已经
// 提过这个" cue. Bearer attached so wiki returns this user's pending hits.
//
// Default search (/api/galgame/search) stays anonymous-only — first-time
// visitors and SSR don't want "突然在首页看到自己的 pending" UX.
func (h *SubmissionHandler) SearchWithPending(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	q := collectQuery(c)
	q.Set("include_pending", "true")
	data, appErr := h.svc.SearchWithPending(c.Context(), token, q)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}
