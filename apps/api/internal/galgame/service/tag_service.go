package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type TagService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewTagService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *TagService {
	return &TagService{wikiClient: wikiClient, enricher: enricher}
}

type wikiTagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
}

type wikiTagListResp struct {
	Items []wikiTagListItem `json:"items"`
	Total int64             `json:"total"`
}

type wikiTagDetail struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Alias       []dto.WikiAlias `json:"alias"`
}

type wikiTagDetailResp struct {
	Tag      wikiTagDetail         `json:"tag"`
	Galgames []dto.WikiGalgameItem `json:"galgames"`
	Total    int64                 `json:"total"`
}

// GetList — GET /galgame-tag
func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.TagListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/tag", rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiTagListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	tags := make([]dto.TagListItem, len(parsed.Items))
	for i, t := range parsed.Items {
		tags[i] = dto.TagListItem{
			ID: t.ID, Name: t.Name, Category: t.Category,
			GalgameCount: t.GalgameCount,
		}
	}
	return &dto.TagListPage{Tags: tags, Total: parsed.Total}, nil
}

// GetDetail — GET /galgame-tag/:name
func (s *TagService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/tag/"+name, rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiTagDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Galgames, isSFW)

	t := parsed.Tag
	return &dto.TagDetail{
		ID:           t.ID,
		Name:         t.Name,
		Category:     t.Category,
		Description:  t.Description,
		Alias:        aliasesToNames(t.Alias),
		Galgame:      s.enricher.ToCards(filtered),
		GalgameCount: parsed.Total,
	}, nil
}
