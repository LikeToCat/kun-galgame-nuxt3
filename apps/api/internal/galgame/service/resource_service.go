package service

import (
	"context"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
)

type ResourceService struct {
	resourceRepo *repository.ResourceRepository
	wikiClient   *client.GalgameClient
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	wikiClient *client.GalgameClient,
) *ResourceService {
	return &ResourceService{resourceRepo: resourceRepo, wikiClient: wikiClient}
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
	userMap := s.resourceRepo.FindUsersByIDs(userIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		card := rowToCard(r, userMap[r.UserID])
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

	userMap := s.resourceRepo.FindUsersByIDs([]int{row.UserID})
	ownerUser := userMap[row.UserID]

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
	userMap := s.resourceRepo.FindUsersByIDs([]int{row.UserID})

	detail := rowToDownloadDetail(row, links, isLiked, userMap[row.UserID])
	return &detail, nil
}

// ──────────────────────────────────────────
// GetGalgameResources — GET /galgame/:gid/resource/all
// ──────────────────────────────────────────

func (s *ResourceService) GetGalgameResources(
	req *dto.GalgameResourcesRequest,
) ([]dto.ResourceCard, *errors.AppError) {
	rows := s.resourceRepo.FindByGalgameID(req.GalgameID)

	userIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
	}
	userMap := s.resourceRepo.FindUsersByIDs(userIDs)

	cards := make([]dto.ResourceCard, len(rows))
	for i, r := range rows {
		cards[i] = rowToCard(r, userMap[r.UserID])
	}
	return cards, nil
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
	userMap := s.resourceRepo.FindUsersByIDs(userIDs)
	briefMap := s.fetchWikiBriefs(ctx, []int{galgameID})

	cards := make([]dto.ResourceCard, len(rows))
	for i, r := range rows {
		card := rowToCard(r, userMap[r.UserID])
		if b, ok := briefMap[galgameID]; ok {
			card.GalgameName = briefToName(b)
		}
		cards[i] = card
	}
	return cards
}
