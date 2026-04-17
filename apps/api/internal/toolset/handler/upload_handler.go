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

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

// UploadSmall generates a presigned PUT URL for files <= 50MB.
// POST /api/toolset/:id/upload/small
func (h *UploadHandler) UploadSmall(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.UploadInitRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.uploadService.InitSmall(c.Context(), id, user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// UploadLarge initiates a multipart upload for files > 50MB and <= 2GB.
// POST /api/toolset/:id/upload/large
func (h *UploadHandler) UploadLarge(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的工具 ID"))
	}

	var req dto.UploadInitRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.uploadService.InitLarge(c.Context(), id, user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// UploadComplete completes a multipart upload and verifies size.
// POST /api/toolset/:id/upload/complete
func (h *UploadHandler) UploadComplete(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UploadCompleteRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.uploadService.Complete(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// UploadAbort aborts a multipart upload and cleans up cache.
// POST /api/toolset/:id/upload/abort
func (h *UploadHandler) UploadAbort(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UploadAbortRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.uploadService.Abort(c.Context(), &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "上传已取消")
}
