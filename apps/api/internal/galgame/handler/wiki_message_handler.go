package handler

import (
	"encoding/json"
	"log/slog"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// WikiMessageHandler exposes the wiki notification stream to kungal users
// and admins, plus the kungal-local read-state cursor.
type WikiMessageHandler struct {
	svc *service.WikiMessageService
}

func NewWikiMessageHandler(svc *service.WikiMessageService) *WikiMessageHandler {
	return &WikiMessageHandler{svc: svc}
}

// MessagesMine — GET /api/galgame/messages/mine (any authenticated user)
func (h *WikiMessageHandler) MessagesMine(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.MessagesMine(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// AdminMessages — GET /api/admin/galgame/messages (role >= 2)
// Caller must already be in a RequireRole(2)-gated route group.
func (h *WikiMessageHandler) AdminMessages(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.AdminMessages(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// ReadStateRequest matches the PUT body for /api/galgame/messages/read-state.
type ReadStateRequest struct {
	LastReadMessageID int64 `json:"last_read_message_id"`
}

// GetReadState — GET /api/galgame/messages/read-state
//
// Returns the cursor as { last_read_message_id: <int64> }. Frontend uses
// this together with the /messages/mine list to compute unread counts.
func (h *WikiMessageHandler) GetReadState(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	last, err := h.svc.GetReadState(user.UID)
	if err != nil {
		slog.Warn("查询 wiki 消息已读游标失败", "uid", user.UID, "error", err)
		return response.Error(c, errors.ErrInternal("查询失败"))
	}
	return response.OK(c, fiber.Map{"last_read_message_id": last})
}

// SetReadState — PUT /api/galgame/messages/read-state
func (h *WikiMessageHandler) SetReadState(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req ReadStateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求体格式错误"))
	}
	if req.LastReadMessageID < 0 {
		return response.Error(c, errors.ErrBadRequest("last_read_message_id 不能为负"))
	}

	if err := h.svc.SetReadState(user.UID, req.LastReadMessageID); err != nil {
		slog.Warn("更新 wiki 消息已读游标失败",
			"uid", user.UID, "last_id", req.LastReadMessageID, "error", err)
		return response.Error(c, errors.ErrInternal("更新失败"))
	}
	return response.OKMessage(c, "已读状态已更新")
}
