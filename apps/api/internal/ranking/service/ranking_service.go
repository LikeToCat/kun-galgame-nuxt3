package service

import (
	"context"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/ranking/dto"
	"kun-galgame-api/internal/ranking/repository"
)

type RankingService struct {
	repo   *repository.RankingRepository
	wikiGC *galgameClient.GalgameClient
}

func NewRankingService(
	repo *repository.RankingRepository,
	gc *galgameClient.GalgameClient,
) *RankingService {
	return &RankingService{repo: repo, wikiGC: gc}
}

// GetGalgameRanking composes galgame ranking rows by
// 1) querying local interaction columns, 2) batch-fetching wiki metadata,
// 3) batch-fetching user info.
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
	users := s.repo.FindUsersByIDs(userIDs)
	userMap := make(map[int]repository.UserInfoRow, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

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

// GetTopicRanking returns topic ranking items.
func (s *RankingService) GetTopicRanking(req *dto.TopicRankingRequest) []dto.TopicRankingItem {
	rows := s.repo.FindTopicRanking(req.SortField, req.SortOrder, req.Page, req.Limit)
	items := make([]dto.TopicRankingItem, len(rows))
	for i, r := range rows {
		items[i] = dto.TopicRankingItem{
			ID:        r.ID,
			Title:     r.Title,
			User:      dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Value:     r.Value,
			SortField: req.SortField,
		}
	}
	return items
}

// GetUserRanking returns user ranking items.
func (s *RankingService) GetUserRanking(req *dto.UserRankingRequest) []dto.UserRankingItem {
	rows := s.repo.FindUserRanking(req.SortField, req.SortOrder, req.Page, req.Limit)
	items := make([]dto.UserRankingItem, len(rows))
	for i, r := range rows {
		items[i] = dto.UserRankingItem{
			ID: r.ID, Name: r.Name, Avatar: r.Avatar,
			Bio: r.Bio, Value: r.Value,
		}
	}
	return items
}
