package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type ToolsetService struct {
	toolsetRepo      *repository.ToolsetRepository
	resourceRepo     *repository.ResourceRepository
	commentRepo      *repository.CommentRepository
	practicalityRepo *repository.PracticalityRepository
	s3               *storage.S3Client

	// Service-level helpers
	practicalitySvc *PracticalityService
	commentSvc      *CommentService
}

func NewToolsetService(
	toolsetRepo *repository.ToolsetRepository,
	resourceRepo *repository.ResourceRepository,
	commentRepo *repository.CommentRepository,
	practicalityRepo *repository.PracticalityRepository,
	s3 *storage.S3Client,
	practicalitySvc *PracticalityService,
	commentSvc *CommentService,
) *ToolsetService {
	return &ToolsetService{
		toolsetRepo:      toolsetRepo,
		resourceRepo:     resourceRepo,
		commentRepo:      commentRepo,
		practicalityRepo: practicalityRepo,
		s3:               s3,
		practicalitySvc:  practicalitySvc,
		commentSvc:       commentSvc,
	}
}

// ──────────────────────────────────────────
// GetList — GET /toolset
// ──────────────────────────────────────────

func (s *ToolsetService) GetList(req *dto.ToolsetListRequest) ([]dto.ToolsetCard, int64) {
	filters := repository.ListFilters{
		Type:     req.Type,
		Language: req.Language,
		Platform: req.Platform,
		Version:  req.Version,
	}
	total := s.toolsetRepo.CountFiltered(filters)

	opts := repository.ListOptions{
		SortField: allowedSortField(req.SortField),
		SortOrder: sortOrder(req.SortOrder),
		Offset:    (req.Page - 1) * req.Limit,
		Limit:     req.Limit,
	}
	toolsets := s.toolsetRepo.ListFiltered(filters, opts)

	toolsetIDs := make([]int, len(toolsets))
	userIDs := make([]int, len(toolsets))
	for i, t := range toolsets {
		toolsetIDs[i] = t.ID
		userIDs[i] = t.UserID
	}

	avgMap := s.practicalityRepo.AveragesForToolsets(toolsetIDs)
	dlMap := s.resourceRepo.DownloadSumsForToolsets(toolsetIDs)
	ccMap := s.commentRepo.CountsForToolsets(toolsetIDs)
	userMap := s.toolsetRepo.FindUsersByIDs(userIDs)

	cards := make([]dto.ToolsetCard, 0, len(toolsets))
	for _, t := range toolsets {
		cards = append(cards, toolsetCardFromRow(t, userMap, avgMap, dlMap, ccMap))
	}

	return cards, total
}

// ──────────────────────────────────────────
// Create — POST /toolset
// ──────────────────────────────────────────

func (s *ToolsetService) Create(
	userID int,
	req *dto.CreateToolsetRequest,
) (*dto.CreatedToolsetResponse, *errors.AppError) {
	homepageJSON, _ := json.Marshal(req.Homepage)

	var toolset model.GalgameToolset
	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		toolset = model.GalgameToolset{
			Name:        req.Name,
			Description: req.Description,
			Type:        req.Type,
			Language:    req.Language,
			Platform:    req.Platform,
			Homepage:    homepageJSON,
			Version:     req.Version,
			UserID:      userID,
		}
		if err := s.toolsetRepo.Create(tx, &toolset); err != nil {
			return err
		}

		// Aliases (trim + skip empties)
		s.toolsetRepo.ReplaceAliases(tx, toolset.ID, trimNonEmpty(req.Aliases))

		// Creator → contributor
		s.toolsetRepo.AddContributor(tx, toolset.ID, userID)

		// Moemoepoint +3
		adjustMoemoepoint(tx, userID, 3)

		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建工具失败")
	}

	return &toolset, nil
}

// ──────────────────────────────────────────
// GetDetail — GET /toolset/:id
// ──────────────────────────────────────────

func (s *ToolsetService) GetDetail(id int) (*dto.ToolsetDetailResponse, *errors.AppError) {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	// View +1 async
	go s.toolsetRepo.IncrementView(id)

	descriptionHTML := markdown.Render(toolset.Description)
	aliases := s.toolsetRepo.FindAliases(id)
	user := s.toolsetRepo.FindUser(toolset.UserID)

	practicality := s.practicalitySvc.Summary(id)
	downloadSum := s.resourceRepo.DownloadSum(id)
	comments := s.commentSvc.GetLatestForDetail(id, 5)
	contributors := s.toolsetRepo.FindContributors(id)
	resources := s.resourceRepo.FindByToolset(id)

	return &dto.ToolsetDetailResponse{
		Toolset:         *toolset,
		DescriptionHTML: descriptionHTML,
		Aliases:         aliases,
		User:            user,
		Practicality:    *practicality,
		DownloadSum:     downloadSum,
		Comments:        comments,
		Contributors:    contributors,
		Resources:       resources,
	}, nil
}

// ──────────────────────────────────────────
// Update — PUT /toolset/:id
// ──────────────────────────────────────────

func (s *ToolsetService) Update(
	userID, userRole, id int,
	req *dto.UpdateToolsetRequest,
) *errors.AppError {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return errors.ErrNotFound("未找到该工具")
	}
	if toolset.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限编辑此工具")
	}

	homepageJSON, _ := json.Marshal(req.Homepage)
	now := time.Now()

	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.toolsetRepo.UpdateFields(tx, id, map[string]any{
			"name":        req.Name,
			"description": req.Description,
			"type":        req.Type,
			"language":    req.Language,
			"platform":    req.Platform,
			"homepage":    homepageJSON,
			"version":     req.Version,
			"edited":      now,
		})
		s.toolsetRepo.ReplaceAliases(tx, id, trimNonEmpty(req.Aliases))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新工具失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete — DELETE /toolset/:id
// ──────────────────────────────────────────

func (s *ToolsetService) Delete(userID, userRole, id int) *errors.AppError {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return errors.ErrNotFound("未找到该工具")
	}
	if toolset.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限删除此工具")
	}

	txErr := s.toolsetRepo.DB().Transaction(func(tx *gorm.DB) error {
		// S3 cleanup: delete all s3 resources (best-effort).
		if s.s3 != nil {
			for _, r := range s.resourceRepo.FindS3ByToolsetTx(tx, id) {
				if r.Code == "" {
					continue
				}
				if err := s.s3.Delete(context.Background(), r.Code); err != nil {
					slog.Warn("删除 S3 资源失败", "key", r.Code, "error", err)
				}
			}
		}

		// Delete related records
		s.toolsetRepo.DeleteAllRelated(tx, id)
		// Delete toolset itself
		s.toolsetRepo.DeleteByID(tx, id)

		// Moemoepoint -3 on the owner
		adjustMoemoepoint(tx, toolset.UserID, -3)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除工具失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

// trimNonEmpty trims whitespace from each string and drops empty ones.
func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
