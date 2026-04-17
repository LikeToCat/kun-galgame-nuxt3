package handler

import (
	"kun-galgame-api/internal/home/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type HomeHandler struct {
	homeService *service.HomeService
}

func NewHomeHandler(homeService *service.HomeService) *HomeHandler {
	return &HomeHandler{homeService: homeService}
}

// GetHome returns homepage data: galgames + topics.
// GET /api/home
func (h *HomeHandler) GetHome(c *fiber.Ctx) error {
	isSFW := utils.IsSFW(c)

	resp, err := h.homeService.GetHome(c.Context(), isSFW)
	if err != nil {
		return response.Error(c, errors.ErrInternal("获取首页数据失败"))
	}
	return response.OK(c, resp)
}
