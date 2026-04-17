package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/toolset/dto"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────
// Constants
// ──────────────────────────────────────────

const (
	MaxSmallFileSize = 50 * 1024 * 1024       // 50MB
	MaxLargeFileSize = 2 * 1024 * 1024 * 1024 // 2GB
	ChunkSize        = 5 * 1024 * 1024        // 5MB
	UploadTTL        = 3600 * time.Second
	PresignExpires   = 3600 * time.Second
)

var allowedArchiveExts = map[string]bool{
	".7z": true, ".zip": true, ".rar": true,
}

// ──────────────────────────────────────────
// Redis cache entry
// ──────────────────────────────────────────

// uploadCacheEntry is stored in Redis while an upload is in progress.
type uploadCacheEntry struct {
	Key      string `json:"key"`
	Type     string `json:"type"` // "small" or "multipart"
	Salt     string `json:"salt"`
	FileSize int64  `json:"filesize"`
	Base     string `json:"base"`
	Ext      string `json:"ext"`
	UploadID string `json:"upload_id,omitempty"` // multipart only
}

// ──────────────────────────────────────────
// Service
// ──────────────────────────────────────────

type UploadService struct {
	s3  *storage.S3Client
	rdb *redis.Client
	db  *gorm.DB
}

func NewUploadService(s3 *storage.S3Client, rdb *redis.Client, db *gorm.DB) *UploadService {
	return &UploadService{s3: s3, rdb: rdb, db: db}
}

// ──────────────────────────────────────────
// InitSmall — POST /toolset/:id/upload/small
// ──────────────────────────────────────────

func (s *UploadService) InitSmall(
	ctx context.Context,
	toolsetID, userID int,
	req *dto.UploadInitRequest,
) (*dto.UploadSmallResponse, *errors.AppError) {
	if req.FileSize > MaxSmallFileSize {
		return nil, errors.ErrBadRequest("小文件上传大小不能超过 50MB")
	}

	ext, base, appErr := parseArchiveFilename(req.Filename)
	if appErr != nil {
		return nil, appErr
	}

	salt := generateSalt()
	key := buildS3Key(toolsetID, userID, base, salt, ext)

	presignedURL, err := s.s3.PresignPutObject(ctx, key, req.ContentType, PresignExpires)
	if err != nil {
		return nil, errors.ErrInternal("生成上传链接失败")
	}

	entry := uploadCacheEntry{
		Key:      key,
		Type:     "small",
		Salt:     salt,
		FileSize: req.FileSize,
		Base:     base,
		Ext:      ext,
	}
	if err := s.cacheEntry(ctx, entry); err != nil {
		return nil, errors.ErrInternal("缓存上传信息失败")
	}

	return &dto.UploadSmallResponse{
		PresignedURL: presignedURL,
		Salt:         salt,
		Key:          key,
	}, nil
}

// ──────────────────────────────────────────
// InitLarge — POST /toolset/:id/upload/large
// ──────────────────────────────────────────

func (s *UploadService) InitLarge(
	ctx context.Context,
	toolsetID, userID int,
	req *dto.UploadInitRequest,
) (*dto.UploadLargeResponse, *errors.AppError) {
	if req.FileSize > MaxLargeFileSize {
		return nil, errors.ErrBadRequest("文件大小不能超过 2GB")
	}

	ext, base, appErr := parseArchiveFilename(req.Filename)
	if appErr != nil {
		return nil, appErr
	}

	salt := generateSalt()
	key := buildS3Key(toolsetID, userID, base, salt, ext)

	uploadID, err := s.s3.CreateMultipartUpload(ctx, key, req.ContentType)
	if err != nil {
		return nil, errors.ErrInternal("创建分片上传失败")
	}

	numParts := int((req.FileSize + ChunkSize - 1) / ChunkSize)
	parts := make([]dto.UploadLargePart, 0, numParts)
	for i := 1; i <= numParts; i++ {
		partURL, err := s.s3.PresignUploadPart(ctx, key, uploadID, int32(i), PresignExpires)
		if err != nil {
			// Best-effort abort on failure
			s.s3.AbortMultipartUpload(ctx, key, uploadID)
			return nil, errors.ErrInternal("生成分片上传链接失败")
		}
		parts = append(parts, dto.UploadLargePart{
			PartNumber:   i,
			PresignedURL: partURL,
		})
	}

	entry := uploadCacheEntry{
		Key:      key,
		Type:     "multipart",
		Salt:     salt,
		FileSize: req.FileSize,
		Base:     base,
		Ext:      ext,
		UploadID: uploadID,
	}
	if err := s.cacheEntry(ctx, entry); err != nil {
		// Abort the multipart upload we created so we don't leak it
		s.s3.AbortMultipartUpload(ctx, key, uploadID)
		return nil, errors.ErrInternal("缓存上传信息失败")
	}

	return &dto.UploadLargeResponse{
		UploadID: uploadID,
		Salt:     salt,
		Key:      key,
		Parts:    parts,
	}, nil
}

// ──────────────────────────────────────────
// Complete — POST /toolset/:id/upload/complete
// ──────────────────────────────────────────

func (s *UploadService) Complete(
	ctx context.Context,
	userID int,
	req *dto.UploadCompleteRequest,
) (*dto.UploadCompleteResponse, *errors.AppError) {
	entry, appErr := s.loadEntry(ctx, req.Salt)
	if appErr != nil {
		return nil, appErr
	}

	if entry.Type == "multipart" {
		if len(req.Parts) == 0 {
			return nil, errors.ErrBadRequest("分片信息不能为空")
		}
		completed := make([]types.CompletedPart, 0, len(req.Parts))
		for _, p := range req.Parts {
			etag := p.ETag
			pn := p.PartNumber
			completed = append(completed, types.CompletedPart{
				ETag:       &etag,
				PartNumber: &pn,
			})
		}
		if err := s.s3.CompleteMultipartUpload(context.Background(), entry.Key, entry.UploadID, completed); err != nil {
			return nil, errors.ErrInternal("完成分片上传失败")
		}
	}

	// Verify size via HeadObject (best-effort).
	actualSize, err := s.s3.HeadObject(context.Background(), entry.Key)
	if err != nil {
		slog.Warn("HeadObject 失败", "key", entry.Key, "error", err)
	} else if actualSize != entry.FileSize {
		slog.Warn("文件大小不匹配",
			"expected", entry.FileSize, "actual", actualSize, "key", entry.Key)
	}

	// Update daily upload count
	s.db.Model(&userModel.User{}).Where("id = ?", userID).
		Update("daily_toolset_upload_count", gorm.Expr("daily_toolset_upload_count + 1"))

	// Clean up Redis cache
	s.rdb.Del(ctx, cacheKey(req.Salt))

	return &dto.UploadCompleteResponse{
		Key:  entry.Key,
		Size: actualSize,
	}, nil
}

// ──────────────────────────────────────────
// Abort — POST /toolset/:id/upload/abort
// ──────────────────────────────────────────

func (s *UploadService) Abort(
	ctx context.Context,
	req *dto.UploadAbortRequest,
) *errors.AppError {
	entry, appErr := s.loadEntry(ctx, req.Salt)
	if appErr != nil {
		return appErr
	}

	// Abort multipart upload if applicable
	if entry.Type == "multipart" && entry.UploadID != "" {
		if err := s.s3.AbortMultipartUpload(context.Background(), entry.Key, entry.UploadID); err != nil {
			slog.Warn("中止分片上传失败", "key", entry.Key, "error", err)
		}
	}

	// For small uploads, try to delete the object in case it was already uploaded.
	if entry.Type == "small" {
		s.s3.Delete(context.Background(), entry.Key)
	}

	// Clean up cache
	s.rdb.Del(ctx, cacheKey(req.Salt))

	return nil
}

// ──────────────────────────────────────────
// Redis helpers
// ──────────────────────────────────────────

func cacheKey(salt string) string {
	return "toolset:upload:" + salt
}

func (s *UploadService) cacheEntry(ctx context.Context, entry uploadCacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, cacheKey(entry.Salt), string(data), UploadTTL).Err()
}

func (s *UploadService) loadEntry(ctx context.Context, salt string) (*uploadCacheEntry, *errors.AppError) {
	val, err := s.rdb.Get(ctx, cacheKey(salt)).Result()
	if err != nil {
		return nil, errors.ErrBadRequest("上传会话不存在或已过期")
	}
	var entry uploadCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, errors.ErrInternal("解析上传缓存失败")
	}
	return &entry, nil
}

// ──────────────────────────────────────────
// File name / salt helpers
// ──────────────────────────────────────────

// parseArchiveFilename extracts the lower-cased extension and base-name from a
// filename and validates the extension against the allow-list.
func parseArchiveFilename(filename string) (ext, base string, appErr *errors.AppError) {
	ext = strings.ToLower(filepath.Ext(filename))
	if !allowedArchiveExts[ext] {
		return "", "", errors.ErrBadRequest("仅支持 .7z, .zip, .rar 格式")
	}
	base = strings.TrimSuffix(filename, ext)
	return ext, base, nil
}

func generateSalt() string {
	b := make([]byte, 4) // 4 bytes → 8 hex chars, we take 7
	rand.Read(b)
	return hex.EncodeToString(b)[:7]
}

func buildS3Key(toolsetID, userID int, base, salt, ext string) string {
	return fmt.Sprintf("toolset/%d/%d_%s_%s%s", toolsetID, userID, base, salt, ext)
}
