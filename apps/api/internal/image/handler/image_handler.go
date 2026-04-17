package handler

import (
	"kun-galgame-api/internal/image/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type ImageHandler struct {
	imageService *service.ImageService
}

func NewImageHandler(imageService *service.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// UploadTopicImage handles topic image upload.
// POST /api/image/topic
func (h *ImageHandler) UploadTopicImage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > service.MaxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	key, sErr := h.imageService.UploadTopicImage(c.Context(), user.UID, f, file.Filename)
	if sErr != nil {
		return response.Error(c, sErr)
	}
	return response.OK(c, key)
}
