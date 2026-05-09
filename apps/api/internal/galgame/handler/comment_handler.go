package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
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

// GetComments returns paginated comments for a galgame.
// GET /api/galgame/:gid/comment/all
func (h *CommentHandler) GetComments(c *fiber.Ctx) error {
	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result := h.commentService.GetComments(c.Context(), gid, req.Page, req.Limit)
	return response.Paginated(c, result.Items, result.Total)
}

// CreateComment creates a galgame comment.
// POST /api/galgame/:gid/comment
func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Content      string `json:"content" validate:"required,min=1,max=1007"`
		TargetUserID *int   `json:"target_user_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resp, appErr := h.commentService.CreateComment(c.Context(), user.UID, gid, req.Content, req.TargetUserID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, resp)
}

// DeleteComment deletes a galgame comment.
// DELETE /api/galgame/:gid/comment
func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `query:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.DeleteComment(user.UID, user.Role, req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "评论已删除")
}

// ToggleCommentLike toggles like on a galgame comment.
// PUT /api/galgame/:gid/comment/like
func (h *CommentHandler) ToggleCommentLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `json:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.ToggleCommentLike(user.UID, req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}
