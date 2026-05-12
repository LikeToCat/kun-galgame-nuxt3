package service

import (
	"context"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/userclient"
)

// GalgameEnricher turns wiki galgame items into the enriched GalgameCard
// shape the frontend consumes, applying NSFW filtering and fusing in local
// interaction counts + user info.
//
// This is the single source of truth for "wiki galgame + local enrichment"
// across series / official / engine / tag detail endpoints.
type GalgameEnricher struct {
	galgameRepo *repository.GalgameRepository
	userClient  *userclient.Client
}

func NewGalgameEnricher(galgameRepo *repository.GalgameRepository, userClient *userclient.Client) *GalgameEnricher {
	return &GalgameEnricher{galgameRepo: galgameRepo, userClient: userClient}
}

// FilterSFW removes NSFW items when the caller requests SFW-only content.
func (e *GalgameEnricher) FilterSFW(items []dto.WikiGalgameItem, isSFW bool) []dto.WikiGalgameItem {
	if !isSFW {
		return items
	}
	out := make([]dto.WikiGalgameItem, 0, len(items))
	for _, g := range items {
		if g.ContentLimit == "sfw" {
			out = append(out, g)
		}
	}
	return out
}

// HasNSFW reports whether any item in the list is nsfw.
func (e *GalgameEnricher) HasNSFW(items []dto.WikiGalgameItem) bool {
	for _, g := range items {
		if g.ContentLimit == "nsfw" {
			return true
		}
	}
	return false
}

// Samples returns up to `n` minimal samples (name + banner).
func (e *GalgameEnricher) Samples(items []dto.WikiGalgameItem, n int) []dto.GalgameSample {
	if n > len(items) {
		n = len(items)
	}
	out := make([]dto.GalgameSample, 0, n)
	for i := 0; i < n; i++ {
		g := items[i]
		out = append(out, dto.GalgameSample{
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
			Banner: g.Banner,
		})
	}
	return out
}

// ToCards converts wiki galgame items into enriched GalgameCard DTOs, batch-
// loading users (from OAuth) and local stats once.
func (e *GalgameEnricher) ToCards(ctx context.Context, items []dto.WikiGalgameItem) []dto.GalgameCard {
	if len(items) == 0 {
		return []dto.GalgameCard{}
	}

	galgameIDs := make([]int, len(items))
	userIDs := make([]int, len(items))
	for i, g := range items {
		galgameIDs[i] = g.ID
		userIDs[i] = g.UserID
	}

	userMap := e.userClient.Hydrate(ctx, userIDs)
	localMap := e.galgameRepo.FindLocalBatch(galgameIDs)

	cards := make([]dto.GalgameCard, len(items))
	for i, g := range items {
		cards[i] = dto.GalgameCard{
			ID: g.ID,
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
			Banner:             g.Banner,
			User:               userBriefToDTO(userMap[g.UserID]),
			ContentLimit:       g.ContentLimit,
			// View is a kungal-local stat (each site has its own audience),
			// not metadata; pull from the local stats row instead of wiki.
			View:               localMap[g.ID].View,
			LikeCount:          localMap[g.ID].LikeCount,
			ResourceUpdateTime: g.ResourceUpdateTime,
			Platform:           []string{},
			Language:           []string{},
		}
	}
	return cards
}
