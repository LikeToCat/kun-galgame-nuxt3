package service

import (
	"context"

	"kun-galgame-api/internal/section/dto"
	"kun-galgame-api/internal/section/repository"
	"kun-galgame-api/pkg/userclient"
)

type SectionService struct {
	repo       *repository.SectionRepository
	userClient *userclient.Client
}

func NewSectionService(
	repo *repository.SectionRepository,
	userClient *userclient.Client,
) *SectionService {
	return &SectionService{repo: repo, userClient: userClient}
}

// GetSectionTopics returns topics filtered by section name. Identity is
// hydrated from OAuth via userclient since the repo no longer joins on the
// user table; banned authors are dropped from the listing.
func (s *SectionService) GetSectionTopics(ctx context.Context, req *dto.SectionTopicsRequest) *dto.SectionTopicsResponse {
	rows, total := s.repo.FindSectionTopics(req.Section, req.SortOrder, req.Page, req.Limit)

	uids := userclient.CollectIDs(rows, func(r repository.SectionTopicRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.SectionTopicItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.SectionTopicItem{
			ID: r.ID, Title: r.Title, Content: r.Content,
			View: r.View, LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			HasBestAnswer: r.BestAnswerID != nil, IsNSFW: r.IsNSFW,
			User:    dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.Created,
		})
	}

	return &dto.SectionTopicsResponse{Topics: items, Total: total}
}

// GetCategoryStats returns section stats (topic count + view count + latest topic)
// filtered by category.
func (s *SectionService) GetCategoryStats(category string) []dto.SectionStat {
	rows := s.repo.FindCategoryStats(category)

	stats := make([]dto.SectionStat, len(rows))
	for i, r := range rows {
		stats[i] = dto.SectionStat{
			ID:         r.SectionID,
			Name:       r.SectionName,
			TopicCount: r.TopicCount,
			ViewCount:  r.ViewCount,
		}
		if latest := s.repo.FindLatestTopicInSection(r.SectionID, category); latest != nil {
			stats[i].LatestTopic = &dto.LatestTopic{
				ID:      latest.ID,
				Title:   latest.Title,
				Created: latest.Created,
			}
		}
	}
	return stats
}
