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
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type ToolsetService struct {
	toolsetRepo      *repository.ToolsetRepository
	resourceRepo     *repository.ResourceRepository
	commentRepo      *repository.CommentRepository
	practicalityRepo *repository.PracticalityRepository
	s3               *storage.S3Client
	userClient       *userclient.Client

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
	userClient *userclient.Client,
	practicalitySvc *PracticalityService,
	commentSvc *CommentService,
) *ToolsetService {
	return &ToolsetService{
		toolsetRepo:      toolsetRepo,
		resourceRepo:     resourceRepo,
		commentRepo:      commentRepo,
		practicalityRepo: practicalityRepo,
		s3:               s3,
		userClient:       userClient,
		practicalitySvc:  practicalitySvc,
		commentSvc:       commentSvc,
	}
}

// userBriefFromClient maps a userclient.User to the UserBrief shape used
// across toolset DTOs. Centralized so every call site is consistent.
func userBriefFromClient(u userclient.User) userModel.UserBrief {
	return userModel.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

// ──────────────────────────────────────────
// GetList — GET /toolset
// ──────────────────────────────────────────

func (s *ToolsetService) GetList(ctx context.Context, req *dto.ToolsetListRequest) ([]dto.ToolsetCard, int64) {
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
	userMap := s.userClient.Hydrate(ctx, userIDs)

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

func (s *ToolsetService) GetDetail(ctx context.Context, id int) (*dto.ToolsetDetailResponse, *errors.AppError) {
	toolset, err := s.toolsetRepo.FindByID(id)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	// View +1 async
	go s.toolsetRepo.IncrementView(id)

	descriptionHTML := markdown.Render(toolset.Description)
	aliases := s.toolsetRepo.FindAliases(id)

	practicality := s.practicalitySvc.Summary(id)
	downloadSum := s.resourceRepo.DownloadSum(id)
	commentCount := s.commentRepo.CountByToolset(id)
	comments := s.commentSvc.GetLatestForDetail(ctx, id, 5)
	contributorIDs := s.toolsetRepo.FindContributorIDs(id)
	resources := s.resourceRepo.FindByToolset(id)

	// Hydrate the owner + every contributor in one batch (the owner usually
	// is a contributor too — userclient dedups).
	allUIDs := append([]int{toolset.UserID}, contributorIDs...)
	userMap := s.userClient.Hydrate(ctx, allUIDs)
	user := userBriefFromClient(userMap[toolset.UserID])
	contributors := make([]userModel.UserBrief, 0, len(contributorIDs))
	for _, uid := range contributorIDs {
		contributors = append(contributors, userBriefFromClient(userMap[uid]))
	}

	// homepage is jsonb (RawMessage) on the model; flatten to []string for the
	// frontend. Tolerate null/garbage by falling back to an empty slice.
	homepage := []string{}
	if len(toolset.Homepage) > 0 {
		_ = json.Unmarshal(toolset.Homepage, &homepage)
		if homepage == nil {
			homepage = []string{}
		}
	}

	aliasNames := make([]string, len(aliases))
	for i, a := range aliases {
		aliasNames[i] = a.Name
	}

	resourceItems := make([]dto.ToolsetResourceItem, len(resources))
	for i, r := range resources {
		resourceItems[i] = dto.ToolsetResourceItem{
			ID: r.ID, Type: r.Type, Size: r.Size,
			Download: r.Download, Status: r.Status,
		}
	}

	// practicalityAvg is null when no one has rated yet (matches nitro).
	var avg *float64
	practicalityCount := int64(0)
	for _, c := range practicality.Counts {
		practicalityCount += c
	}
	if practicalityCount > 0 {
		v := practicality.Avg
		avg = &v
	}

	return &dto.ToolsetDetailResponse{
		ID:                 toolset.ID,
		Name:               toolset.Name,
		ContentMarkdown:    toolset.Description,
		ContentHTML:        descriptionHTML,
		Type:               toolset.Type,
		Platform:           toolset.Platform,
		Language:           toolset.Language,
		Version:            toolset.Version,
		Homepage:           homepage,
		View:               toolset.View,
		Download:           downloadSum,
		User:               user,
		Aliases:            aliasNames,
		PracticalityAvg:    avg,
		PracticalityCount:  practicalityCount,
		RatingCounts:       practicality.Counts,
		ResourceUpdateTime: toolset.ResourceUpdateTime,
		Resource:           resourceItems,
		Edited:             toolset.Edited,
		Created:            toolset.CreatedAt,
		Updated:            toolset.UpdatedAt,
		CommentCount:       commentCount,
		CommentPreview:     comments,
		Contributors:       contributors,
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
