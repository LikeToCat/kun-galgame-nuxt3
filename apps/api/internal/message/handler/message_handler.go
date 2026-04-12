package handler

import (
	"strconv"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// GetMessages returns paginated message list.
// GET /api/message
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.ListMessagesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.messageService.GetMessages(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, result)
}

// DeleteMessage deletes a single message.
// DELETE /api/message/:id
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的消息 ID"))
	}

	if appErr := h.messageService.DeleteMessage(c.Context(), user.UID, id); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "消息已删除")
}

// GetSystemMessages returns all system broadcast messages.
// GET /api/message/admin
func (h *MessageHandler) GetSystemMessages(c *fiber.Ctx) error {
	messages, appErr := h.messageService.GetSystemMessages(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, messages)
}

// MarkAdminRead marks all system messages as read.
// PUT /api/message/admin/read
func (h *MessageHandler) MarkAdminRead(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.messageService.MarkAllSystemRead(c.Context()); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "已标记全部已读")
}

// GetNavSummary returns message and system message summary for nav bar.
// GET /api/message/nav/system
func (h *MessageHandler) GetNavSummary(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.messageService.GetNavSummary(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, result)
}

// MarkAllRead marks all user notification messages as read.
// PUT /api/message/system/read
func (h *MessageHandler) MarkAllRead(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.messageService.MarkAllRead(c.Context(), user.UID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "已标记全部已读")
}
