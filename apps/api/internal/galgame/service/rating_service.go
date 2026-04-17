package service

import (
	"context"
	"encoding/json"
	"fmt"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
)

type RatingService struct {
	ratingRepo *repository.RatingRepository
	wikiClient *client.GalgameClient
}

func NewRatingService(
	ratingRepo *repository.RatingRepository,
	wikiClient *client.GalgameClient,
) *RatingService {
	return &RatingService{ratingRepo: ratingRepo, wikiClient: wikiClient}
}

// ──────────────────────────────────────────
// GetAllRatings — GET /galgame-rating/all
// ──────────────────────────────────────────

func (s *RatingService) GetAllRatings(
	ctx context.Context,
	req *dto.RatingListRequest,
) (*dto.RatingListPage, *errors.AppError) {
	// Normalise sort order
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, total := s.ratingRepo.ListPaginated(model.RatingFilter{
		SpoilerLevel: req.SpoilerLevel,
		PlayStatus:   req.PlayStatus,
		GalgameType:  req.GalgameType,
		SortField:    req.SortField,
		SortOrder:    sortOrder,
		Page:         req.Page,
		Limit:        req.Limit,
	})

	// Batch resolve users and galgames
	userIDs := make([]int, len(rows))
	galgameIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		galgameIDs[i] = r.GalgameID
	}
	userMap := s.ratingRepo.FindUsersByIDs(userIDs)
	briefMap := s.fetchWikiBriefs(ctx, galgameIDs)

	cards := make([]dto.RatingCard, len(rows))
	for i, r := range rows {
		cards[i] = ratingRowToCard(r, userMap[r.UserID], briefMap[r.GalgameID])
	}

	return &dto.RatingListPage{RatingData: cards, Total: total}, nil
}

// ──────────────────────────────────────────
// GetRatingDetail — GET /galgame-rating/:id
// ──────────────────────────────────────────

func (s *RatingService) GetRatingDetail(
	ctx context.Context,
	ratingID, currentUserID int,
) (*dto.RatingDetail, *errors.AppError) {
	row, ok := s.ratingRepo.FindByID(ratingID)
	if !ok {
		return nil, errors.ErrNotFound("评分不存在")
	}

	// Fire-and-forget view increment; reflect it in response too.
	s.ratingRepo.IncrementView(ratingID)
	row.View++

	// Liked users
	likerIDs := s.ratingRepo.FindLikerIDs(ratingID)
	likedUsers := s.ratingRepo.FindUsersListByIDs(likerIDs)
	isLiked := containsInt(likerIDs, currentUserID)

	// Author + comments
	authorMap := s.ratingRepo.FindUsersByIDs([]int{row.UserID})
	commentRows := s.ratingRepo.FindComments(ratingID)
	comments := make([]dto.RatingCommentItem, len(commentRows))
	for i, cm := range commentRows {
		comments[i] = ratingCommentRowToDTO(cm)
	}

	// Galgame detail from wiki
	galgame := s.buildRatingGalgame(ctx, row.GalgameID)

	// Authored user projection
	authorBriefs := make([]dto.UserBrief, 0, len(likedUsers))
	for _, u := range likedUsers {
		authorBriefs = append(authorBriefs, userBriefToDTO(u))
	}

	detail := &dto.RatingDetail{
		ID:           row.ID,
		User:         userBriefToDTO(authorMap[row.UserID]),
		Recommend:    row.Recommend,
		Overall:      row.Overall,
		View:         row.View,
		GalgameType:  rawJSON(row.GalgameType),
		PlayStatus:   row.PlayStatus,
		ShortSummary: row.ShortSummary,
		SpoilerLevel: row.SpoilerLevel,
		RatingScores: rowToScores(row),
		LikeCount:    len(likerIDs),
		IsLiked:      isLiked,
		LikedUsers:   authorBriefs,
		Comments:     comments,
		Created:      row.Created,
		Updated:      row.Updated,
		Galgame:      galgame,
	}
	return detail, nil
}

// ──────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────

func (s *RatingService) fetchWikiBriefs(
	ctx context.Context,
	galgameIDs []int,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.wikiClient.GetBatch(ctx, galgameIDs)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

// buildRatingGalgame fetches galgame detail from wiki and computes rating stats.
func (s *RatingService) buildRatingGalgame(ctx context.Context, galgameID int) dto.RatingGalgameDetail {
	summary := dto.RatingGalgameDetail{
		ID:       galgameID,
		Official: []dto.RatingOfficial{},
	}

	data, err := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%d", galgameID), nil)
	if err == nil {
		var envelope dto.WikiGalgameDetailResponse
		if jsonErr := json.Unmarshal(data, &envelope); jsonErr == nil {
			g := envelope.Galgame
			summary.ID = g.ID
			summary.Banner = g.Banner
			summary.ContentLimit = g.ContentLimit
			summary.AgeLimit = g.AgeLimit
			summary.OriginalLanguage = g.OriginalLanguage
			summary.Name = dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			}
			summary.Official = wikiOfficialsToDTO(g.Official)
		}
	}

	sum, count := s.ratingRepo.GalgameRatingStats(galgameID)
	summary.Rating = sum
	summary.RatingCount = count
	return summary
}
