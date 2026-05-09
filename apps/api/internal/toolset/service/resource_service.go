package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type ResourceService struct {
	resourceRepo *repository.ResourceRepository
	toolsetRepo  *repository.ToolsetRepository
	s3           *storage.S3Client
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	toolsetRepo *repository.ToolsetRepository,
	s3 *storage.S3Client,
) *ResourceService {
	return &ResourceService{
		resourceRepo: resourceRepo,
		toolsetRepo:  toolsetRepo,
		s3:           s3,
	}
}

// ──────────────────────────────────────────
// GetResourceDetail — GET /toolset/:id/resource/detail
// ──────────────────────────────────────────

func (s *ResourceService) GetResourceDetail(
	req *dto.ResourceDetailRequest,
) (*dto.ResourceDetailResponse, *errors.AppError) {
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	// Download +1 (fire-and-forget)
	go s.resourceRepo.IncrementDownload(resource.ID)

	user := s.resourceRepo.FindUser(resource.UserID)

	return &dto.ResourceDetailResponse{
		Resource: *resource,
		User:     user,
	}, nil
}

// ──────────────────────────────────────────
// CreateResource — POST /toolset/:id/resource
// ──────────────────────────────────────────

func (s *ResourceService) CreateResource(
	userID, toolsetID int,
	req *dto.CreateResourceRequest,
) (*dto.CreatedResourceResponse, *errors.AppError) {
	// Verify toolset exists
	if _, err := s.toolsetRepo.FindByID(toolsetID); err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	var resource model.GalgameToolsetResource
	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		resource = model.GalgameToolsetResource{
			Content:   req.Content,
			Type:      req.Type,
			Code:      req.Code,
			Password:  req.Password,
			Size:      req.Size,
			Note:      req.Note,
			ToolsetID: toolsetID,
			UserID:    userID,
		}
		if err := s.resourceRepo.Create(tx, &resource); err != nil {
			return err
		}

		// Moemoepoint +3
		adjustMoemoepoint(tx, userID, 3)

		// Add contributor (ignore duplicate)
		s.toolsetRepo.AddContributor(tx, toolsetID, userID)

		// Refresh resource_update_time
		s.toolsetRepo.UpdateResourceTime(tx, toolsetID, time.Now())

		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建资源失败")
	}

	return &resource, nil
}

// ──────────────────────────────────────────
// UpdateResource — PUT /toolset/:id/resource
// ──────────────────────────────────────────

func (s *ResourceService) UpdateResource(
	userID, userRole int,
	req *dto.UpdateResourceRequest,
) *errors.AppError {
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return errors.ErrNotFound("未找到该资源")
	}

	if resource.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限编辑此资源")
	}

	now := time.Now()
	updates := map[string]any{
		"password": req.Password,
		"note":     req.Note,
		"edited":   now,
	}

	// S3 type: only password and note can be updated.
	// User type: all fields can be updated.
	if resource.Type == "user" {
		updates["content"] = req.Content
		updates["code"] = req.Code
		updates["size"] = req.Size
	}

	s.resourceRepo.UpdateFields(resource, updates)
	return nil
}

// ──────────────────────────────────────────
// DeleteResource — DELETE /toolset/:id/resource
// ──────────────────────────────────────────

func (s *ResourceService) DeleteResource(
	userID, userRole int,
	req *dto.DeleteResourceRequest,
) *errors.AppError {
	resource, err := s.resourceRepo.FindByID(req.ResourceID)
	if err != nil {
		return errors.ErrNotFound("未找到该资源")
	}

	if resource.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限删除此资源")
	}

	// S3 cleanup (best-effort)
	if resource.Type == "s3" && resource.Code != "" && s.s3 != nil {
		if err := s.s3.Delete(context.Background(), resource.Code); err != nil {
			slog.Warn("删除 S3 资源失败", "key", resource.Code, "error", err)
		}
	}

	s.resourceRepo.Delete(resource)

	// Moemoepoint -3 on the resource owner
	adjustMoemoepoint(s.resourceRepo.DB(), resource.UserID, -3)

	return nil
}

// ──────────────────────────────────────────
// Shared helpers
// ──────────────────────────────────────────

// adjustMoemoepoint atomically bumps the user's moemoepoint by delta.
// Duplicated here (rather than imported from galgame/service) to avoid a
// cross-module cycle; see internal/galgame/service/interaction.go for the
// original pattern.
func adjustMoemoepoint(db *gorm.DB, userID, delta int) {
	if userID <= 0 || delta == 0 {
		return
	}
	db.Model(&userModel.KungalUserState{}).Where("user_id = ?", userID).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", delta))
}
