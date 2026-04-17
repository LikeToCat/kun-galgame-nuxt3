package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type OfficialService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewOfficialService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *OfficialService {
	return &OfficialService{wikiClient: wikiClient, enricher: enricher}
}

// ──────────────────────────────────────────
// Wiki response shapes
// ──────────────────────────────────────────

type wikiOfficialListItem struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	Link         string           `json:"link"`
	Category     string           `json:"category"`
	Lang         string           `json:"lang"`
	Alias        []dto.WikiAlias  `json:"alias"`
	GalgameCount int              `json:"galgame_count"`
}

type wikiOfficialListResp struct {
	Items []wikiOfficialListItem `json:"items"`
	Total int64                  `json:"total"`
}

type wikiOfficialDetail struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Link        string          `json:"link"`
	Category    string          `json:"category"`
	Lang        string          `json:"lang"`
	Description string          `json:"description"`
	Alias       []dto.WikiAlias `json:"alias"`
}

type wikiOfficialDetailResp struct {
	Official wikiOfficialDetail    `json:"official"`
	Galgames []dto.WikiGalgameItem `json:"galgames"`
	Total    int64                 `json:"total"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-official
// ──────────────────────────────────────────

func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.OfficialListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/official", rawQuery)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiOfficialListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	items := make([]dto.OfficialListItem, len(parsed.Items))
	for i, o := range parsed.Items {
		items[i] = dto.OfficialListItem{
			ID:           o.ID,
			Name:         o.Name,
			Link:         o.Link,
			Category:     o.Category,
			Lang:         o.Lang,
			Alias:        aliasesToNames(o.Alias),
			GalgameCount: o.GalgameCount,
		}
	}
	return &dto.OfficialListPage{Officials: items, Total: parsed.Total}, nil
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame-official/:name
// ──────────────────────────────────────────

func (s *OfficialService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.OfficialDetail, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/official/"+name, rawQuery)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiOfficialDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Galgames, isSFW)

	o := parsed.Official
	return &dto.OfficialDetail{
		ID:           o.ID,
		Name:         o.Name,
		Link:         o.Link,
		Category:     o.Category,
		Lang:         o.Lang,
		Description:  o.Description,
		Alias:        aliasesToNames(o.Alias),
		Galgame:      s.enricher.ToCards(filtered),
		GalgameCount: parsed.Total,
	}, nil
}

// aliasesToNames extracts the name field from a slice of WikiAlias.
func aliasesToNames(aliases []dto.WikiAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}
