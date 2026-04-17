package handler

import (
	"kun-galgame-api/internal/activity/dto"
	"kun-galgame-api/internal/activity/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type ActivityHandler struct {
	activityService *service.ActivityService
}

func NewActivityHandler(activityService *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

// GetActivity returns activity feed filtered by type.
// GET /api/activity
func (h *ActivityHandler) GetActivity(c *fiber.Ctx) error {
	var req dto.ActivityRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.activityService.GetActivity(c.Context(), req.Type, req.Page, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, res.Items, res.Total)
}

// GetTimeline returns mixed activity timeline.
// GET /api/activity/timeline
func (h *ActivityHandler) GetTimeline(c *fiber.Ctx) error {
	var req dto.TimelineRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.activityService.GetTimeline(c.Context(), req.Page, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, res.Items, res.Total)
}
