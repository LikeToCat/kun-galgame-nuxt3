package handler

import (
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/errors"
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

// CreateComment creates a comment on a reply.
// POST /api/topic/:tid/comment
func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.CreateComment(
		c.Context(), user.UID,
		req.TopicID, req.ReplyID, req.TargetUserID, req.Content,
	); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "评论发表成功")
}

// ToggleCommentLike toggles like on a comment.
// PUT /api/topic/:tid/comment/like
func (h *CommentHandler) ToggleCommentLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CommentInteractionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.ToggleCommentLike(c.Context(), user.UID, req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}

// DeleteComment deletes a comment.
// DELETE /api/topic/:tid/comment
func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	commentID, err := strconv.Atoi(c.Query("commentId"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的评论 ID"))
	}

	if appErr := h.commentService.DeleteComment(c.Context(), user.UID, user.Role, commentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "评论已删除")
}
