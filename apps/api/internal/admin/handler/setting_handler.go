package handler

import (
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type SettingHandler struct {
	settingService *service.SettingService
}

func NewSettingHandler(settingService *service.SettingService) *SettingHandler {
	return &SettingHandler{settingService: settingService}
}

// GetRegisterSetting returns whether registration is enabled.
// GET /api/admin/setting/register
func (h *SettingHandler) GetRegisterSetting(c *fiber.Ctx) error {
	resp := h.settingService.GetRegisterSetting(c.Context())
	return response.OK(c, resp)
}

// ToggleRegisterSetting toggles the registration on/off.
// PUT /api/admin/setting/register
func (h *SettingHandler) ToggleRegisterSetting(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	h.settingService.ToggleRegisterSetting(c.Context())
	return response.OKMessage(c, "注册设置已更新")
}
