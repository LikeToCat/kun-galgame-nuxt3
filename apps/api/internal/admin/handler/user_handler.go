package handler

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetUserList returns paginated user list for admin.
// GET /api/admin/user
func (h *UserHandler) GetUserList(c *fiber.Ctx) error {
	var req dto.GetUserListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resp := h.userService.GetUserList(req.Page, req.Limit)
	return response.OK(c, resp)
}

// SearchUsers searches users by name for admin.
// GET /api/admin/user/search
func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return response.Error(c, errors.ErrBadRequest("搜索关键词不能为空"))
	}

	users := h.userService.SearchUsers(q)
	return response.OK(c, users)
}
