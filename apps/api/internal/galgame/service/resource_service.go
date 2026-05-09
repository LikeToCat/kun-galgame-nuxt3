package service

import (
	"context"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
)

type ResourceService struct {
	resourceRepo *repository.ResourceRepository
	wikiClient   *client.GalgameClient
	userClient   *userclient.Client
	helpers      InteractionHelpers
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
) *ResourceService {
	return &ResourceService{resourceRepo: resourceRepo, wikiClient: wikiClient, userClient: userClient}
}

// ──────────────────────────────────────────
// GetResourceList — GET /galgame-resource
// ──────────────────────────────────────────

func (s *ResourceService) GetResourceList(
	ctx context.Context,
	req *dto.ResourceListRequest,
) (*dto.ResourceListPage, *errors.AppError) {
	total := s.resourceRepo.CountAll()
	rows := s.resourceRepo.ListPaginated(req.Page, req.Limit)

	galgameIDs, userIDs := collectIDs(rows)
	briefMap := s.fetchWikiBriefs(ctx, galgameIDs)
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := rowToCard(r, u)
		if b, ok := briefMap[r.GalgameID]; ok {
			card.GalgameName = briefToName(b)
		}
		cards = append(cards, card)
	}

	return &dto.ResourceListPage{Resources: cards, Total: total}, nil
}

// ──────────────────────────────────────────
// GetResourceDetail — GET /galgame-resource/:id
// Returns (detail, nil) on success or (nil, nil) for "not found" (legacy format).
// ──────────────────────────────────────────

// NotFoundSentinel is returned by GetResourceDetail when the resource doesn't exist.
// The handler serialises it to the JSON string "not found" for backwards compat.
type ResourceNotFound struct{}

func (s *ResourceService) GetResourceDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDetailPage, *ResourceNotFound, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, &ResourceNotFound{}, nil
	}

	// Fire-and-forget view increment. We add 1 to the returned value too so
	// the client sees the freshly-incremented count without re-fetching.
	s.resourceRepo.IncrementView(resourceID)
	row.View++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)

	ownerUser, _, _ := s.userClient.User(ctx, row.UserID)

	resource := rowToDownloadDetail(row, links, isLiked, ownerUser)

	// Galgame summary
	galgameSummary := s.buildGalgameSummary(ctx, row.GalgameID)

	// Recommendations (max 6)
	recRows := s.resourceRepo.FindRecommendations(row.GalgameID, resourceID, 6)
	recommendations := s.buildRecommendations(ctx, recRows, row.GalgameID)

	return &dto.ResourceDetailPage{
		Galgame:         galgameSummary,
		Resource:        resource,
		Recommendations: recommendations,
	}, nil, nil
}

// ──────────────────────────────────────────
// GetResourceDownloadDetail — GET /galgame-resource/:id/detail
// Bumps the download counter and returns links/code/password.
// ──────────────────────────────────────────

func (s *ResourceService) GetResourceDownloadDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDownloadDetail, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	s.resourceRepo.IncrementDownload(resourceID)
	row.Download++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)
	owner, _, _ := s.userClient.User(ctx, row.UserID)

	detail := rowToDownloadDetail(row, links, isLiked, owner)
	return &detail, nil
}

// ──────────────────────────────────────────
// GetGalgameResources — GET /galgame/:gid/resource/all
// ──────────────────────────────────────────

func (s *ResourceService) GetGalgameResources(
	ctx context.Context,
	req *dto.GalgameResourcesRequest,
) ([]dto.ResourceCard, *errors.AppError) {
	rows := s.resourceRepo.FindByGalgameID(req.GalgameID)

	userIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		cards = append(cards, rowToCard(r, u))
	}
	return cards, nil
}

// ──────────────────────────────────────────
// GetRecommendations — GET /galgame-resource/:id/recommend
// Returns up to 6 sibling resources sorted by like_count, sharing the same galgame.
// ──────────────────────────────────────────

func (s *ResourceService) GetRecommendations(
	ctx context.Context,
	resourceID int,
) ([]dto.ResourceCard, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该资源")
	}
	recRows := s.resourceRepo.FindRecommendations(row.GalgameID, resourceID, 6)
	return s.buildRecommendations(ctx, recRows, row.GalgameID), nil
}

// ──────────────────────────────────────────
// CreateResource — POST /galgame/:gid/resource
// Creates the resource row + links + provider tags in a single transaction,
// awarding +3 moemoepoint and bumping local resource_count.
// ──────────────────────────────────────────

func (s *ResourceService) CreateResource(
	uid int,
	req *dto.CreateGalgameResourceRequest,
) *errors.AppError {
	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	res := &model.GalgameResource{
		Type:      req.Type,
		Language:  req.Language,
		Platform:  req.Platform,
		Size:      req.Size,
		Code:      req.Code,
		Password:  req.Password,
		Note:      req.Note,
		GalgameID: req.GalgameID,
		UserID:    uid,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.Create(tx, res); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, res.ID, providers); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviderNames(tx, res.ID, providerNames); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, res.ID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.AdjustLocalResourceCount(tx, req.GalgameID, 1); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, uid, constants.RewardCreateResource)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("创建 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// UpdateResource — PUT /galgame/:gid/resource
// Replaces all links, recomputes providers, patches scalar fields.
// ──────────────────────────────────────────

func (s *ResourceService) UpdateResource(
	uid, role int,
	req *dto.UpdateGalgameResourceRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if row.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限更新这个 Galgame 资源")
	}

	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	fields := map[string]any{
		"type":     req.Type,
		"language": req.Language,
		"platform": req.Platform,
		"size":     req.Size,
		"code":     req.Code,
		"password": req.Password,
		"note":     req.Note,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateFields(tx, req.GalgameResourceID, fields); err != nil {
			return err
		}
		if err := s.resourceRepo.DeleteLinks(tx, req.GalgameResourceID); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, req.GalgameResourceID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, req.GalgameResourceID, providers); err != nil {
			return err
		}
		return s.resourceRepo.ReplaceProviderNames(tx, req.GalgameResourceID, providerNames)
	})
	if txErr != nil {
		return errors.ErrInternal("更新 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// DeleteResource — DELETE /galgame/:gid/resource
// Deducts 5 + like_count moemoepoint from the original uploader and decrements
// the galgame resource_count. Cascade deletes links/likes via FK.
// ──────────────────────────────────────────

func (s *ResourceService) DeleteResource(
	uid, role, resourceID int,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除这个 Galgame 资源")
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.helpers.AdjustMoemoepoint(tx, row.UserID, -(row.LikeCount + 5))
		if err := s.resourceRepo.DeleteByID(tx, resourceID); err != nil {
			return err
		}
		return s.resourceRepo.AdjustLocalResourceCount(tx, row.GalgameID, -1)
	})
	if txErr != nil {
		return errors.ErrInternal("删除 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// ToggleLike — PUT /galgame/:gid/resource/like
// Self-like is rejected. Toggles the like row + maintains like_count and a +/-1
// moemoepoint swing on the liker (matches existing nitro behaviour).
// ──────────────────────────────────────────

func (s *ResourceService) ToggleLike(
	uid int,
	req *dto.ToggleResourceLikeRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到该资源")
	}
	if row.UserID == uid {
		return errors.ErrBadRequest("您不能给自己的资源点赞")
	}

	links := s.resourceRepo.FindLinks(req.GalgameResourceID)
	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, has := s.resourceRepo.FindLike(tx, req.GalgameResourceID, uid)
		var delta int
		if has {
			if err := s.resourceRepo.DeleteLike(tx, existing); err != nil {
				return err
			}
			delta = -1
		} else {
			if err := s.resourceRepo.CreateLike(tx, req.GalgameResourceID, uid); err != nil {
				return err
			}
			delta = 1
		}
		if err := s.resourceRepo.AdjustLikeCount(tx, req.GalgameResourceID, delta); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, uid, delta)
		s.helpers.CreateGalgameMessageWithContent(
			tx, uid, row.UserID, "liked", preview, row.GalgameID,
		)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// ──────────────────────────────────────────
// MarkValid / MarkExpired — PUT valid|expired
// MarkValid is restricted to the original uploader; MarkExpired is open.
// MarkExpired sends a non-deduped notification to the uploader.
// ──────────────────────────────────────────

func (s *ResourceService) MarkValid(uid int, resourceID int) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok || row.UserID != uid {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if err := s.resourceRepo.UpdateStatus(s.resourceRepo.DB(), resourceID, 0); err != nil {
		return errors.ErrInternal("更新失败")
	}
	return nil
}

func (s *ResourceService) MarkExpired(uid int, resourceID int) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.Status == 1 {
		return errors.ErrBadRequest("该资源已经被标记为失效")
	}

	links := s.resourceRepo.FindLinks(resourceID)
	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateStatus(tx, resourceID, 1); err != nil {
			return err
		}
		s.helpers.CreateGalgameMessageWithContent(
			tx, uid, row.UserID, "expired", preview, row.GalgameID,
		)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────

func (s *ResourceService) fetchWikiBriefs(
	ctx context.Context,
	galgameIDs []int,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	briefMap, _ := s.wikiClient.GetBatch(ctx, galgameIDs)
	if briefMap == nil {
		return map[int]client.GalgameBrief{}
	}
	return briefMap
}

func (s *ResourceService) buildGalgameSummary(
	ctx context.Context,
	galgameID int,
) dto.ResourceGalgameSummary {
	summary := dto.ResourceGalgameSummary{
		ID:       galgameID,
		Platform: []string{}, Language: []string{}, Type: []string{},
	}

	briefMap := s.fetchWikiBriefs(ctx, []int{galgameID})
	b, ok := briefMap[galgameID]
	if !ok {
		return summary
	}

	aggs := s.resourceRepo.AggregateByGalgame(galgameID)
	platforms, languages, types := collectAggregate(aggs)
	localView := s.resourceRepo.FindGalgameView(galgameID)

	return dto.ResourceGalgameSummary{
		ID:                 b.ID,
		Name:               briefToName(b),
		Banner:             b.Banner,
		ContentLimit:       b.ContentLimit,
		View:               localView,
		ResourceUpdateTime: b.ResourceUpdateTime,
		OriginalLanguage:   b.OriginalLanguage,
		AgeLimit:           b.AgeLimit,
		Platform:           platforms,
		Language:           languages,
		Type:               types,
	}
}

func (s *ResourceService) buildRecommendations(
	ctx context.Context,
	rows []model.GalgameResourceRow,
	galgameID int,
) []dto.ResourceCard {
	userIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchWikiBriefs(ctx, []int{galgameID})

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := rowToCard(r, u)
		if b, ok := briefMap[galgameID]; ok {
			card.GalgameName = briefToName(b)
		}
		cards = append(cards, card)
	}
	return cards
}
