package service

import (
	"context"
	"encoding/json"
	"maps"
	"net/url"
	"strconv"

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
// Search — GET /galgame-official/search
// ──────────────────────────────────────────
//
// Wiki search is Meilisearch-backed and returns the standard
// `{items, total, processing_time_ms}` envelope. The alias field on each
// item may be missing entirely or populated with {id, name, ...} objects;
// aliasesToNames(nil) → []string{} keeps the frontend contract intact.
//
// The frontend (galgame/official/Container.vue) does
//   `searchResult.value = res`  expecting a bare GalgameOfficialItem[].
// We unwrap `items` here so the gateway response stays compatible without
// touching the frontend.
func (s *OfficialService) Search(
	ctx context.Context,
	rawQuery url.Values,
) ([]dto.OfficialListItem, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/official/search", rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []wikiOfficialListItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}
	raw := resp.Items

	items := make([]dto.OfficialListItem, len(raw))
	for i, o := range raw {
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
	return items, nil
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
	data, appErr := s.wikiClient.Get(ctx, "/official/"+name, withSFWFilter(rawQuery, isSFW))
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
		Galgame:      s.enricher.ToCards(ctx, filtered),
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

// withSFWFilter clones q and, when isSFW is true, injects content_limit=sfw
// so the wiki service filters NSFW galgames server-side and returns a
// matching total. A nil/empty q is handled.
func withSFWFilter(q url.Values, isSFW bool) url.Values {
	out := url.Values{}
	maps.Copy(out, q)
	if isSFW {
		out.Set("content_limit", "sfw")
	}
	return out
}

// atoiOr parses s as an int, returning fallback on any failure (empty / bad).
func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
