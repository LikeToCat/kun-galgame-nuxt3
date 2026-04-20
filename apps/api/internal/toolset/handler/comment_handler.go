package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/service"
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

// GetComments returns paginated comments for a toolset.
// GET /api/toolset/:id/comment/all
//
// Response shape: { commentData: ToolsetCommentItem[], total: number } —
// matches the legacy nitro contract that the frontend Container.vue reads.
func (h *CommentHandler) GetComments(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.CommentListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	resp := h.commentService.GetComments(id, &req)
	return response.OK(c, resp)
}

// CreateComment creates a comment on a toolset.
// POST /api/toolset/:id/comment
func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.CreateCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	comment, appErr := h.commentService.CreateComment(user.UID, id, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, comment)
}

// UpdateComment edits a comment (owner only).
// PUT /api/toolset/:id/comment
func (h *CommentHandler) UpdateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.UpdateComment(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论更新成功")
}

// DeleteComment deletes a comment.
// DELETE /api/toolset/:id/comment
func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.DeleteCommentRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.DeleteComment(user.UID, user.Role, id, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论已删除")
}
