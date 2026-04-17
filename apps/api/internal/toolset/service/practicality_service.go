package service

import (
	"math"

	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/errors"
)

type PracticalityService struct {
	practicalityRepo *repository.PracticalityRepository
}

func NewPracticalityService(practicalityRepo *repository.PracticalityRepository) *PracticalityService {
	return &PracticalityService{practicalityRepo: practicalityRepo}
}

// GetPracticality returns the rating distribution for a toolset, including
// the current user's rating (if any).
func (s *PracticalityService) GetPracticality(toolsetID, currentUserID int) *dto.PracticalityResponse {
	summary := s.Summary(toolsetID)

	var mine *int
	if currentUserID > 0 {
		if p, _ := s.practicalityRepo.FindUserRating(toolsetID, currentUserID); p != nil {
			rate := p.Rate
			mine = &rate
		}
	}

	return &dto.PracticalityResponse{
		Counts: summary.Counts,
		Avg:    summary.Avg,
		Mine:   mine,
	}
}

// Summary returns the (counts, avg) pair for a toolset, used by the detail view.
func (s *PracticalityService) Summary(toolsetID int) *dto.PracticalitySummary {
	counts := map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for _, rc := range s.practicalityRepo.CountsByToolset(toolsetID) {
		counts[rc.Rate] = rc.Count
	}
	avg := s.practicalityRepo.AverageRate(toolsetID)

	return &dto.PracticalitySummary{
		Counts: counts,
		Avg:    math.Round(avg*100) / 100,
	}
}

// UpsertPracticality creates or updates the current user's rating.
func (s *PracticalityService) UpsertPracticality(toolsetID, userID int, req *dto.UpsertPracticalityRequest) *errors.AppError {
	if err := s.practicalityRepo.Upsert(toolsetID, userID, req.Rate); err != nil {
		return errors.ErrInternal("评分失败")
	}
	return nil
}
