package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"kun-galgame-api/internal/image/repository"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/pkg/errors"
)

const (
	MaxImageSize    = 10 * 1024 * 1024 // 10MB
	dailyImageLimit = 50
	imageBedBucket  = "topic"
)

type ImageService struct {
	repo *repository.ImageRepository
	s3   *storage.S3Client
}

func NewImageService(repo *repository.ImageRepository, s3 *storage.S3Client) *ImageService {
	return &ImageService{repo: repo, s3: s3}
}

// UploadTopicImage validates the user's daily quota, decodes + re-encodes
// the image as PNG, uploads it to S3, then increments the daily counter.
// Returns the S3 key (to be prefixed by CDN base URL on the frontend).
func (s *ImageService) UploadTopicImage(ctx context.Context, uid int, r io.Reader, filename string) (string, *errors.AppError) {
	// Check daily limit
	count, err := s.repo.GetDailyCount(uid)
	if err != nil {
		return "", errors.ErrInternal("查询用户失败")
	}
	if count >= dailyImageLimit {
		return "", errors.ErrBadRequest("今日图片上传次数已达上限")
	}

	// Decode image
	img, _, err := image.Decode(r)
	if err != nil {
		return "", errors.ErrBadRequest("无效的图片格式")
	}

	// TODO: resize large images with imaging library.
	// For now, re-encode as PNG (WebP requires cgo; PNG is a safe fallback).
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", errors.ErrInternal("图片处理失败")
	}

	// Upload to S3
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".png"
	}
	key := fmt.Sprintf("%s/user_%d/%d%s", imageBedBucket, uid, uid*1000+count, ext)

	if err := s.s3.Upload(ctx, key, "image/png", bytes.NewReader(buf.Bytes())); err != nil {
		return "", errors.ErrInternal("上传图片失败")
	}

	// Increment daily count
	s.repo.IncrementDailyCount(uid)

	return key, nil
}
