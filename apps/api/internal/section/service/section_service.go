package service

import (
	"kun-galgame-api/internal/section/dto"
	"kun-galgame-api/internal/section/repository"
)

type SectionService struct {
	repo *repository.SectionRepository
}

func NewSectionService(repo *repository.SectionRepository) *SectionService {
	return &SectionService{repo: repo}
}

// GetSectionTopics returns topics filtered by section name.
func (s *SectionService) GetSectionTopics(req *dto.SectionTopicsRequest) *dto.SectionTopicsResponse {
	rows, total := s.repo.FindSectionTopics(req.Section, req.SortOrder, req.Page, req.Limit)

	items := make([]dto.SectionTopicItem, len(rows))
	for i, r := range rows {
		items[i] = dto.SectionTopicItem{
			ID: r.ID, Title: r.Title, Content: r.Content,
			View: r.View, LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			HasBestAnswer: r.BestAnswerID != nil, IsNSFW: r.IsNSFW,
			User:    dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Created: r.Created,
		}
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
