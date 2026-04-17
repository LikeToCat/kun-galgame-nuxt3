package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// GetComments returns nested comments for a website.
// GET /api/website/:domain/comment
func (h *CommentHandler) GetComments(c *fiber.Ctx) error {
	var req dto.CommentListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.commentService.GetComments(req.WebsiteID))
}

// CreateComment creates a website comment.
// POST /api/website/:domain/comment
func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	comment, appErr := h.commentService.CreateComment(user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, comment)
}

// DeleteComment deletes a website comment.
// DELETE /api/website/:domain/comment
func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteCommentRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.DeleteComment(user.UID, user.Role, req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论已删除")
}
