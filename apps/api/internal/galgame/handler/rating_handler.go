package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type RatingHandler struct {
	ratingService *service.RatingService
}

func NewRatingHandler(ratingService *service.RatingService) *RatingHandler {
	return &RatingHandler{ratingService: ratingService}
}

// GetAllRatings returns paginated galgame ratings.
// GET /api/galgame-rating/all
func (h *RatingHandler) GetAllRatings(c *fiber.Ctx) error {
	var req dto.RatingListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.ratingService.GetAllRatings(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetRatingDetail returns a single rating with comments, liked users, and galgame.
// GET /api/galgame-rating/:id
func (h *RatingHandler) GetRatingDetail(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的评分 ID"))
	}

	currentUID := optionalUID(c)
	detail, appErr := h.ratingService.GetRatingDetail(c.Context(), id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// CreateRating — POST /api/galgame-rating
func (h *RatingHandler) CreateRating(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateRatingRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	created, appErr := h.ratingService.CreateRating(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, created)
}

// UpdateRating — PUT /api/galgame-rating/:id
func (h *RatingHandler) UpdateRating(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateRatingRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.UpdateRating(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评分更新成功")
}

// DeleteRating — DELETE /api/galgame-rating/:id
func (h *RatingHandler) DeleteRating(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteRatingRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.DeleteRating(user.UID, user.Role, req.GalgameRatingID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评分已删除")
}

// ToggleLike — PUT /api/galgame-rating/:id/like
func (h *RatingHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ToggleRatingLikeRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.ToggleRatingLike(user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// CreateComment — POST /api/galgame-rating/:id/comment
func (h *RatingHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateRatingCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	created, appErr := h.ratingService.CreateRatingComment(user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, created)
}

// UpdateComment — PUT /api/galgame-rating/:id/comment
func (h *RatingHandler) UpdateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateRatingCommentRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	updated, appErr := h.ratingService.UpdateRatingComment(user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, updated)
}

// DeleteComment — DELETE /api/galgame-rating/:id/comment
func (h *RatingHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteRatingCommentRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.DeleteRatingComment(user.UID, user.Role, req.GalgameRatingCommentID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论已删除")
}
