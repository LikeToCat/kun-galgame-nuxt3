package handler

import (
	"net/url"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// EntityHandler groups the wiki-entity endpoints (series/official/engine/tag).
// All of these proxy to wiki, enrich with local data, and apply NSFW filtering.
type EntityHandler struct {
	seriesService   *service.SeriesService
	officialService *service.OfficialService
	engineService   *service.EngineService
	tagService      *service.TagService
}

func NewEntityHandler(
	series *service.SeriesService,
	official *service.OfficialService,
	engine *service.EngineService,
	tag *service.TagService,
) *EntityHandler {
	return &EntityHandler{
		seriesService:   series,
		officialService: official,
		engineService:   engine,
		tagService:      tag,
	}
}

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

// GetSeriesList — GET /galgame-series
func (h *EntityHandler) GetSeriesList(c *fiber.Ctx) error {
	var req dto.SeriesListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.seriesService.GetList(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetSeriesDetail — GET /galgame-series/:id
func (h *EntityHandler) GetSeriesDetail(c *fiber.Ctx) error {
	detail, appErr := h.seriesService.GetDetail(c.Context(), c.Params("id"), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Official
// ──────────────────────────────────────────

// GetOfficialList — GET /galgame-official
func (h *EntityHandler) GetOfficialList(c *fiber.Ctx) error {
	page, appErr := h.officialService.GetList(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetOfficialDetail — GET /galgame-official/:name
func (h *EntityHandler) GetOfficialDetail(c *fiber.Ctx) error {
	detail, appErr := h.officialService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRename(c, "officialId", "official_id"),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Engine
// ──────────────────────────────────────────

// GetEngineList — GET /galgame-engine
func (h *EntityHandler) GetEngineList(c *fiber.Ctx) error {
	items, appErr := h.engineService.GetList(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetEngineDetail — GET /galgame-engine/:name
func (h *EntityHandler) GetEngineDetail(c *fiber.Ctx) error {
	detail, appErr := h.engineService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRename(c, "engineId", "engine_id"),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Tag
// ──────────────────────────────────────────

// GetTagList — GET /galgame-tag
func (h *EntityHandler) GetTagList(c *fiber.Ctx) error {
	page, appErr := h.tagService.GetList(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetMultiTagGalgames — GET /galgame-tag/multi
func (h *EntityHandler) GetMultiTagGalgames(c *fiber.Ctx) error {
	page, appErr := h.tagService.GetByMultiTag(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetTagDetail — GET /galgame-tag/:name
func (h *EntityHandler) GetTagDetail(c *fiber.Ctx) error {
	detail, appErr := h.tagService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRename(c, "tagId", "tag_id"),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Query helpers
// ──────────────────────────────────────────

// collectQuery converts the Fiber request query args into url.Values.
func collectQuery(c *fiber.Ctx) url.Values {
	q := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		q.Set(string(key), string(value))
	})
	return q
}

// collectQueryWithRename is like collectQuery but renames one key on the way
// through (used to translate camelCase frontend params to wiki's snake_case).
func collectQueryWithRename(c *fiber.Ctx, from, to string) url.Values {
	q := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k == from {
			q.Set(to, string(value))
		} else {
			q.Set(k, string(value))
		}
	})
	return q
}
