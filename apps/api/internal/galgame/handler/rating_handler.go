package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
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
