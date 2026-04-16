package handler

import (
	"strconv"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *service.UserService
	wikiGC      *galgameClient.GalgameClient
}

func NewUserHandler(userService *service.UserService, gc *galgameClient.GalgameClient) *UserHandler {
	return &UserHandler{userService: userService, wikiGC: gc}
}

// GetProfile returns a user's public profile.
// GET /api/user/:uid
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	profile, appErr := h.userService.GetUserProfile(c.Context(), uid)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, profile)
}

// CheckIn handles daily check-in.
// POST /api/user/check-in
func (h *UserHandler) CheckIn(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	points, appErr := h.userService.CheckIn(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, points)
}

// UpdateBio updates the user's bio.
// PUT /api/user/bio
func (h *UserHandler) UpdateBio(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateBioRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateBio(c.Context(), user.UID, req.Bio); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "签名更新成功")
}

// UpdateUsername updates the user's name (costs moemoepoints).
// PUT /api/user/username
func (h *UserHandler) UpdateUsername(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateUsernameRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateUsername(c.Context(), user.UID, req.Username); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户名更新成功")
}

// UpdateEmail updates the user's email after code verification.
// PUT /api/user/email
func (h *UserHandler) UpdateEmail(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateEmailRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	if appErr := h.userService.UpdateEmail(c.Context(), user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "邮箱更新成功")
}

// GetEmail returns the user's masked email.
// GET /api/user/email
func (h *UserHandler) GetEmail(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	email, appErr := h.userService.GetMaskedEmail(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, email)
}

// GetStatus returns the user's status (moemoepoints, check-in, unread messages).
// GET /api/user/status
func (h *UserHandler) GetStatus(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	status, appErr := h.userService.GetUserStatus(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, status)
}

// UploadAvatar handles avatar upload.
// POST /api/user/avatar
func (h *UserHandler) UploadAvatar(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片错误"))
	}

	link, appErr := h.userService.UploadAvatar(c.Context(), user.UID, buf)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, link)
}

// GetUserGalgames returns a user's galgame list.
// GET /api/user/:uid/galgames
func (h *UserHandler) GetUserGalgames(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserGalgamesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	ids, total, appErr := h.userService.GetUserGalgameIDs(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if len(ids) == 0 {
		return response.Paginated(c, []fiber.Map{}, total)
	}

	briefMap, wikiErr := h.wikiGC.GetBatch(c.Context(), ids)
	if wikiErr != nil {
		return response.Paginated(c, []fiber.Map{}, total)
	}

	items := make([]fiber.Map, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		items = append(items, fiber.Map{
			"id":     b.ID,
			"vndbId": b.VndbID,
			"name": fiber.Map{
				"en-us": b.NameEnUs, "ja-jp": b.NameJaJp,
				"zh-cn": b.NameZhCn, "zh-tw": b.NameZhTw,
			},
			"banner":       b.Banner,
			"contentLimit": b.ContentLimit,
		})
	}

	return response.Paginated(c, items, total)
}

// GetUserTopics returns a user's topic list.
// GET /api/user/:uid/topics
func (h *UserHandler) GetUserTopics(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.UserTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, total, appErr := h.userService.GetUserTopics(c.Context(), uid, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.Paginated(c, items, total)
}

// BanUser bans or unbans a user (admin only).
// PUT /api/user/:uid/ban
func (h *UserHandler) BanUser(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	var req dto.BanUserRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.userService.BanUser(c.Context(), uid, req.Status); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户状态更新成功")
}

// DeleteUser permanently deletes a user (admin only).
// DELETE /api/user/:uid
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	uid, err := strconv.Atoi(c.Params("uid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}

	if appErr := h.userService.DeleteUser(c.Context(), uid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "用户已删除")
}
