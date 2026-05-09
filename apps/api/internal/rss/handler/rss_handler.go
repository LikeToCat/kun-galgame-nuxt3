package handler

import (
	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/rss/dto"
	"kun-galgame-api/internal/rss/repository"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v2"
)

// RSSHandler handles RSS feed routes.
// No service layer — logic is a single query with fixed filters.
// Galgame RSS additionally enriches local stub IDs with wiki metadata.
type RSSHandler struct {
	repo       *repository.RSSRepository
	wikiClient *galgameClient.GalgameClient
	userClient *userclient.Client
}

func NewRSSHandler(
	repo *repository.RSSRepository,
	wikiClient *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *RSSHandler {
	return &RSSHandler{repo: repo, wikiClient: wikiClient, userClient: userClient}
}

// GetTopicRSS returns recent topics for RSS feed.
// GET /api/rss/topic
func (h *RSSHandler) GetTopicRSS(c *fiber.Ctx) error {
	rows := h.repo.FindRecentSFWTopics()
	uids := userclient.CollectIDs(rows, func(r dto.TopicRSSItem) int { return r.UserID })
	userMap := h.userClient.Hydrate(c.Context(), uids)
	for i := range rows {
		u := userMap[rows[i].UserID]
		rows[i].UserName = u.Name
	}
	return response.OK(c, rows)
}

// GetGalgameRSS returns the 10 most recent galgames as RSS items.
// GET /api/rss/galgame
//
// Local DB only stores stub IDs + created timestamps — name/banner/user come
// from the wiki batch endpoint. Description is left empty since wiki batch
// doesn't include intros.
func (h *RSSHandler) GetGalgameRSS(c *fiber.Ctx) error {
	rows := h.repo.FindRecentGalgameIDs(10)
	if len(rows) == 0 {
		return response.OK(c, []dto.GalgameRSSItem{})
	}

	ids := make([]int, len(rows))
	createdByID := make(map[int]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		createdByID[r.ID] = r.Created
	}

	briefMap, _ := h.wikiClient.GetBatch(c.Context(), ids)
	if briefMap == nil {
		briefMap = map[int]galgameClient.GalgameBrief{}
	}

	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := h.userClient.Hydrate(c.Context(), userIDs)

	items := make([]dto.GalgameRSSItem, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		u := userMap[b.UserID]
		items = append(items, dto.GalgameRSSItem{
			ID:     id,
			Name:   pickPreferredName(b),
			Banner: b.Banner,
			User: dto.GalgameRSSUser{
				ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			},
			Description: "",
			Created:     createdByID[id],
		})
	}
	return response.OK(c, items)
}

// pickPreferredName mirrors the legacy getPreferredLanguageText fallback chain:
// zh-cn > en-us > ja-jp > zh-tw. Returns the first non-empty entry.
func pickPreferredName(b galgameClient.GalgameBrief) string {
	candidates := []string{b.NameZhCn, b.NameEnUs, b.NameJaJp, b.NameZhTw}
	for _, n := range candidates {
		if n != "" {
			return n
		}
	}
	return ""
}
