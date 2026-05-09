package service

import (
	"context"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/ranking/dto"
	"kun-galgame-api/internal/ranking/repository"
	"kun-galgame-api/pkg/userclient"
)

type RankingService struct {
	repo       *repository.RankingRepository
	wikiGC     *galgameClient.GalgameClient
	userClient *userclient.Client
}

func NewRankingService(
	repo *repository.RankingRepository,
	gc *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *RankingService {
	return &RankingService{repo: repo, wikiGC: gc, userClient: userClient}
}

// GetGalgameRanking composes galgame ranking rows by
// 1) querying local interaction columns, 2) batch-fetching wiki metadata,
// 3) batch-fetching user info from OAuth.
func (s *RankingService) GetGalgameRanking(
	ctx context.Context, req *dto.GalgameRankingRequest,
) []dto.GalgameRankingItem {
	rows := s.repo.FindGalgameLocal(req.SortField, req.SortOrder, req.Page, req.Limit)
	if len(rows) == 0 {
		return []dto.GalgameRankingItem{}
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	briefMap, appErr := s.wikiGC.GetBatch(ctx, ids)
	if appErr != nil {
		return []dto.GalgameRankingItem{}
	}

	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

	items := make([]dto.GalgameRankingItem, 0, len(rows))
	for _, r := range rows {
		b, ok := briefMap[r.ID]
		if !ok {
			continue
		}
		u := userMap[b.UserID]
		items = append(items, dto.GalgameRankingItem{
			ID: r.ID,
			Name: dto.LocaleName{
				EnUS: b.NameEnUs, JaJP: b.NameJaJp,
				ZhCN: b.NameZhCn, ZhTW: b.NameZhTw,
			},
			User:      dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Banner:    b.Banner,
			Value:     r.Value,
			SortField: req.SortField,
		})
	}
	return items
}

// GetTopicRanking returns topic ranking items. Identity is hydrated from OAuth
// via userclient.
func (s *RankingService) GetTopicRanking(ctx context.Context, req *dto.TopicRankingRequest) []dto.TopicRankingItem {
	rows := s.repo.FindTopicRanking(req.SortField, req.SortOrder, req.Page, req.Limit)
	uids := userclient.CollectIDs(rows, func(r repository.TopicRankingRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.TopicRankingItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.TopicRankingItem{
			ID:        r.ID,
			Title:     r.Title,
			User:      dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Value:     r.Value,
			SortField: req.SortField,
		})
	}
	return items
}

// GetUserRanking returns user ranking items. Sorted by kungal_user_state
// column; identity (name/avatar/bio) is hydrated from OAuth via userclient.
func (s *RankingService) GetUserRanking(ctx context.Context, req *dto.UserRankingRequest) []dto.UserRankingItem {
	rows := s.repo.FindUserRanking(req.SortField, req.SortOrder, req.Page, req.Limit)
	uids := userclient.CollectIDs(rows, func(r repository.UserRankingRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.UserRankingItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.UserRankingItem{
			ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			Bio: u.Bio, Value: r.Value,
		})
	}
	return items
}
