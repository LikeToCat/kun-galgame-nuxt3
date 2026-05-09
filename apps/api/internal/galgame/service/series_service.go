package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type SeriesService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewSeriesService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *SeriesService {
	return &SeriesService{wikiClient: wikiClient, enricher: enricher}
}

// ──────────────────────────────────────────
// Wiki response shapes (parsing-only)
// ──────────────────────────────────────────

type wikiSeriesListItem struct {
	ID           int                   `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Galgame      []dto.WikiGalgameItem `json:"galgame"`
	GalgameCount int                   `json:"galgame_count"`
	Created      string                `json:"created"`
	Updated      string                `json:"updated"`
}

type wikiSeriesListResp struct {
	Items []wikiSeriesListItem `json:"items"`
	Total int64                `json:"total"`
}

type wikiSeriesDetail struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Galgame     []dto.WikiGalgameItem `json:"galgame"`
	Created     string                `json:"created"`
	Updated     string                `json:"updated"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-series
// ──────────────────────────────────────────
//
// We pull the whole series set (typically < 200) so we can apply NSFW
// filtering in Go before paginating, which avoids sparse pages when many
// series contain only NSFW titles.

func (s *SeriesService) GetList(
	ctx context.Context,
	req *dto.SeriesListRequest,
	isSFW bool,
) (*dto.SeriesListPage, *errors.AppError) {
	query := url.Values{
		"page": {"1"}, "limit": {"500"}, "include": {"galgame"},
	}
	data, appErr := s.wikiClient.Get(ctx, "/series", query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiSeriesListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	// Backfill galgame arrays where the list endpoint omitted them.
	s.backfillGalgames(ctx, parsed.Items)

	items := make([]dto.SeriesListItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		filtered := s.enricher.FilterSFW(item.Galgame, isSFW)
		if isSFW && len(filtered) == 0 {
			continue
		}

		count := item.GalgameCount
		if isSFW {
			count = len(filtered)
		}

		items = append(items, dto.SeriesListItem{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			IsNSFW:        s.enricher.HasNSFW(filtered),
			SampleGalgame: s.enricher.Samples(filtered, 4),
			GalgameCount:  count,
			Created:       item.Created,
			Updated:       item.Updated,
		})
	}

	total := int64(len(items))
	items = paginate(items, req.Page, req.Limit)
	return &dto.SeriesListPage{Series: items, Total: total}, nil
}

// backfillGalgames fetches galgame data for series entries whose galgame slice
// is empty but galgame_count > 0 (wiki list endpoint sometimes returns sparse).
func (s *SeriesService) backfillGalgames(ctx context.Context, items []wikiSeriesListItem) {
	for i := range items {
		if len(items[i].Galgame) > 0 || items[i].GalgameCount == 0 {
			continue
		}
		data, err := s.wikiClient.Get(ctx, fmt.Sprintf("/series/%d", items[i].ID), nil)
		if err != nil {
			continue
		}
		var parsed struct {
			Galgame []dto.WikiGalgameItem `json:"galgame"`
		}
		if json.Unmarshal(data, &parsed) == nil {
			items[i].Galgame = parsed.Galgame
		}
	}
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame-series/:id
// ──────────────────────────────────────────

func (s *SeriesService) GetDetail(
	ctx context.Context,
	seriesID string,
	isSFW bool,
) (*dto.SeriesDetail, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/series/"+seriesID, nil)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiSeriesDetail
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Galgame, isSFW)

	return &dto.SeriesDetail{
		ID:            parsed.ID,
		Name:          parsed.Name,
		Description:   parsed.Description,
		IsNSFW:        s.enricher.HasNSFW(filtered),
		SampleGalgame: s.enricher.Samples(filtered, 5),
		GalgameCount:  len(filtered),
		Galgame:       s.enricher.ToCards(ctx, filtered),
		Created:       parsed.Created,
		Updated:       parsed.Updated,
	}, nil
}

// paginate slices `items` according to page/limit, clamping out-of-range.
func paginate[T any](items []T, page, limit int) []T {
	start := (page - 1) * limit
	if start >= len(items) {
		return []T{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
