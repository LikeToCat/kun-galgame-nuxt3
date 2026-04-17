package handler

import (
	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// GetNavContact returns the chat room list for the message sidebar.
// GET /api/message/nav/contact
func (h *ChatHandler) GetNavContact(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	items, appErr := h.chatService.GetNavContact(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetChatHistory returns chat message history with a user, chronological ASC.
// GET /api/message/chat/history
func (h *ChatHandler) GetChatHistory(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.GetChatHistoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, appErr := h.chatService.GetChatHistory(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// SendChatMessage sends a private chat message (replaces socket.io path).
// POST /api/message/chat/send
func (h *ChatHandler) SendChatMessage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.SendChatMessageRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.chatService.SendChatMessage(c.Context(), user.UID, user.Name, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "发送成功")
}
