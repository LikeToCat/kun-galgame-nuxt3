package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type EngineService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewEngineService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *EngineService {
	return &EngineService{wikiClient: wikiClient, enricher: enricher}
}

type wikiEngineListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
}

type wikiEngineDetail struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Alias       []string `json:"alias"`
}

type wikiEngineDetailResp struct {
	Engine   wikiEngineDetail      `json:"engine"`
	Galgames []dto.WikiGalgameItem `json:"galgames"`
	Total    int64                 `json:"total"`
}

// GetList — GET /galgame-engine
func (s *EngineService) GetList(ctx context.Context) ([]dto.EngineListItem, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/engine", nil)
	if appErr != nil {
		return nil, appErr
	}
	var engines []wikiEngineListItem
	if err := json.Unmarshal(data, &engines); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	items := make([]dto.EngineListItem, len(engines))
	for i, e := range engines {
		items[i] = dto.EngineListItem{
			ID:           e.ID,
			Name:         e.Name,
			Description:  e.Description,
			Alias:        emptyStrSliceIfNil(e.Alias),
			GalgameCount: e.GalgameCount,
		}
	}
	return items, nil
}

// GetDetail — GET /galgame-engine/:name
func (s *EngineService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.EngineDetail, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/engine/"+name, withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiEngineDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Galgames, isSFW)

	e := parsed.Engine
	return &dto.EngineDetail{
		ID:           e.ID,
		Name:         e.Name,
		Description:  e.Description,
		Alias:        emptyStrSliceIfNil(e.Alias),
		Galgame:      s.enricher.ToCards(ctx, filtered),
		GalgameCount: parsed.Total,
	}, nil
}

func emptyStrSliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
