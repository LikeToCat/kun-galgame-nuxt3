package common

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"path/filepath"
	"strings"

	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/middleware"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	maxImageSize     = 10 * 1024 * 1024 // 10MB
	maxImageWidth    = 1920
	maxImageHeight   = 1080
	dailyImageLimit  = 50
	imageBedBucket   = "topic"
)

type ImageHandler struct {
	db *gorm.DB
	s3 *storage.S3Client
}

func NewImageHandler(db *gorm.DB, s3 *storage.S3Client) *ImageHandler {
	return &ImageHandler{db: db, s3: s3}
}

// UploadTopicImage handles topic image upload.
// POST /api/image/topic
func (h *ImageHandler) UploadTopicImage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Check daily limit
	var u userModel.User
	if err := h.db.Select("daily_image_count").First(&u, user.UID).Error; err != nil {
		return response.Error(c, errors.ErrInternal("查询用户失败"))
	}
	if u.DailyImageCount >= dailyImageLimit {
		return response.Error(c, errors.ErrBadRequest("今日图片上传次数已达上限"))
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > maxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	// Decode image
	img, _, err := image.Decode(f)
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的图片格式"))
	}

	// TODO: resize large images with imaging library
	// For now, upload as-is (the original Nitro code used sharp for WebP conversion)

	// Encode as PNG (WebP requires cgo; PNG is a safe fallback)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return response.Error(c, errors.ErrInternal("图片处理失败"))
	}

	// Upload to S3
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".png"
	}
	key := fmt.Sprintf("%s/user_%d/%d%s", imageBedBucket, user.UID,
		user.UID*1000+u.DailyImageCount, ext)

	if err := h.s3.Upload(c.Context(), key, "image/png", bytes.NewReader(buf.Bytes())); err != nil {
		return response.Error(c, errors.ErrInternal("上传图片失败"))
	}

	// Increment daily count
	h.db.Model(&userModel.User{}).Where("id = ?", user.UID).
		Update("daily_image_count", gorm.Expr("daily_image_count + 1"))

	// Return URL (the S3 endpoint + key)
	imageURL := key // The frontend should prepend the CDN base URL
	return response.OK(c, imageURL)
}
