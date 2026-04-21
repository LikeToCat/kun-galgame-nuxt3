package service

import (
	"context"
	"encoding/json"
	"maps"
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

// wikiMultiTagResp is the {items, total} shape wiki returns for /tag/multi.
type wikiMultiTagResp struct {
	Items []dto.WikiGalgameItem `json:"items"`
	Total int64                 `json:"total"`
}

// TagMultiPage is the enriched response for GET /galgame-tag/multi.
type TagMultiPage struct {
	Galgames []dto.GalgameCard `json:"galgames"`
	Total    int64             `json:"total"`
}

// GetByMultiTag — GET /galgame-tag/multi
//
// Proxies to the wiki /tag/multi endpoint with param renamed to the
// snake_case the wiki expects and content_limit forwarded in SFW mode,
// then enriches the resulting galgames with local like counts etc.
func (s *TagService) GetByMultiTag(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*TagMultiPage, *errors.AppError) {
	q := withSFWFilter(rawQuery, isSFW)
	// tagIds → tag_ids (wiki uses snake_case)
	if v := q.Get("tagIds"); v != "" {
		q.Del("tagIds")
		q.Set("tag_ids", v)
	}

	data, appErr := s.wikiClient.Get(ctx, "/tag/multi", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiMultiTagResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Items, isSFW)
	return &TagMultiPage{
		Galgames: s.enricher.ToCards(filtered),
		Total:    parsed.Total,
	}, nil
}

// GetList — GET /galgame-tag
//
// In SFW mode we drop tags whose category is "sexual". Because the wiki
// service paginates before we can filter, we over-fetch the full tag set
// (typically a few thousand rows) and paginate on the API side so the
// frontend sees an accurate total.
func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagListPage, *errors.AppError) {
	page := atoiOr(rawQuery.Get("page"), 1)
	limit := atoiOr(rawQuery.Get("limit"), 100)

	q := url.Values{}
	maps.Copy(q, rawQuery)
	q.Set("page", "1")
	q.Set("limit", "5000")

	data, appErr := s.wikiClient.Get(ctx, "/tag", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiTagListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := make([]dto.TagListItem, 0, len(parsed.Items))
	for _, t := range parsed.Items {
		if isSFW && t.Category == "sexual" {
			continue
		}
		filtered = append(filtered, dto.TagListItem{
			ID: t.ID, Name: t.Name, Category: t.Category,
			GalgameCount: t.GalgameCount,
		})
	}

	total := int64(len(filtered))
	tags := paginate(filtered, page, limit)
	return &dto.TagListPage{Tags: tags, Total: total}, nil
}

// GetDetail — GET /galgame-tag/:name
//
// In SFW mode we forward content_limit=sfw to the wiki so it filters the
// galgame list server-side and returns a matching total; the local
// FilterSFW call is kept as a defensive net.
func (s *TagService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	q := withSFWFilter(rawQuery, isSFW)
	data, appErr := s.wikiClient.Get(ctx, "/tag/"+name, q)
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
