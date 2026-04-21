package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

// GalgameService handles the "core" galgame lifecycle: create, merge PR,
// detail aggregation, list with filters, and local interaction toggles.
type GalgameService struct {
	galgameRepo       *repository.GalgameRepository
	interactionRepo   *repository.GalgameInteractionRepository
	listRepo          *repository.GalgameListRepository
	resourceMetaRepo  *repository.GalgameResourceMetaRepository
	detailRatingRepo  *repository.GalgameDetailRatingRepository
	wikiClient        *client.GalgameClient
	helpers           InteractionHelpers
}

func NewGalgameService(
	galgameRepo *repository.GalgameRepository,
	interactionRepo *repository.GalgameInteractionRepository,
	listRepo *repository.GalgameListRepository,
	resourceMetaRepo *repository.GalgameResourceMetaRepository,
	detailRatingRepo *repository.GalgameDetailRatingRepository,
	wikiClient *client.GalgameClient,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		interactionRepo:  interactionRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		detailRatingRepo: detailRatingRepo,
		wikiClient:       wikiClient,
	}
}

// ──────────────────────────────────────────
// Create — POST /galgame
// ──────────────────────────────────────────

// Create forwards the payload to wiki, then awards moemoepoint and creates
// the local stub row for the new galgame. Returns the raw wiki response body
// so the handler can forward it verbatim.
func (s *GalgameService) Create(
	ctx context.Context,
	userID int,
	token string,
	body []byte,
) (json.RawMessage, *errors.AppError) {
	data, appErr := s.wikiClient.PostWithToken(ctx, "/galgame", token, json.RawMessage(body))
	if appErr != nil {
		return nil, appErr
	}

	var created dto.WikiCreatedResp
	_ = json.Unmarshal(data, &created)

	if created.ID > 0 {
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.galgameRepo.CreateLocalStub(tx, created.ID)
			s.helpers.AdjustMoemoepoint(tx, userID, constants.RewardCreateGalgame)
			return nil
		})
	}
	return data, nil
}

// ──────────────────────────────────────────
// MergePR — PUT /galgame/:gid/prs/:id/merge
// ──────────────────────────────────────────

func (s *GalgameService) MergePR(
	ctx context.Context,
	mergerID int,
	gid, prID, token string,
) (json.RawMessage, *errors.AppError) {
	// Look up submitter before merging (wiki may purge pending info after merge)
	prData, appErr := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%s/prs/%s", gid, prID), nil)
	if appErr != nil {
		return nil, appErr
	}
	var prInfo dto.WikiPRDetail
	_ = json.Unmarshal(prData, &prInfo)

	data, appErr := s.wikiClient.PutWithToken(
		ctx, fmt.Sprintf("/galgame/%s/prs/%s/merge", gid, prID), token, nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	submitter := prInfo.PR.UserID
	if submitter > 0 && submitter != mergerID {
		gidInt, _ := strconv.Atoi(gid)
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.helpers.AdjustMoemoepoint(tx, submitter, constants.RewardPRMerge)
			s.helpers.CreateGalgameMessage(tx, mergerID, submitter, "merged", gidInt)
			return nil
		})
	}
	return data, nil
}

// ──────────────────────────────────────────
// Interactions — PUT /galgame/:gid/like|favorite
// ──────────────────────────────────────────

// ToggleLike reports an error when the user tries to self-like, otherwise
// atomically flips the like and adjusts owner moemoepoint + notification.
func (s *GalgameService) ToggleLike(
	ctx context.Context,
	userID, galgameID int,
) *errors.AppError {
	ownerID := s.fetchOwnerID(ctx, galgameID)
	if ownerID == userID {
		return errors.ErrBadRequest("您不能给自己点赞")
	}

	s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
		liked := s.interactionRepo.ToggleLike(tx, userID, galgameID)
		if liked {
			s.helpers.AdjustMoemoepoint(tx, ownerID, 1)
			s.helpers.CreateGalgameMessage(tx, userID, ownerID, "liked", galgameID)
		} else {
			s.helpers.AdjustMoemoepoint(tx, ownerID, -1)
		}
		return nil
	})
	return nil
}

// ToggleFavorite flips favorite state and (on +1 direction) rewards the
// galgame owner by +1 moemoe and sends a `favorite` notification — matching
// legacy Nitro behavior. Owner id is resolved via wiki; if the lookup fails
// we still flip the flag but skip moemoe / notification.
func (s *GalgameService) ToggleFavorite(ctx context.Context, userID, galgameID int) *errors.AppError {
	ownerID := s.fetchOwnerID(ctx, galgameID)

	s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
		favorited := s.interactionRepo.ToggleFavorite(tx, userID, galgameID)
		if ownerID == 0 || ownerID == userID {
			return nil
		}
		if favorited {
			s.helpers.AdjustMoemoepoint(tx, ownerID, 1)
			s.helpers.CreateGalgameMessage(tx, userID, ownerID, "favorite", galgameID)
		} else {
			s.helpers.AdjustMoemoepoint(tx, ownerID, -1)
		}
		return nil
	})
	return nil
}

// fetchOwnerID reads the owner user_id from wiki (0 on any failure).
func (s *GalgameService) fetchOwnerID(ctx context.Context, galgameID int) int {
	data, err := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%d", galgameID), nil)
	if err != nil {
		return 0
	}
	var env struct {
		Galgame struct {
			UserID int `json:"user_id"`
		} `json:"galgame"`
	}
	_ = json.Unmarshal(data, &env)
	return env.Galgame.UserID
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame/:gid
// ──────────────────────────────────────────

func (s *GalgameService) GetDetail(
	ctx context.Context,
	galgameID, currentUserID int,
) (*dto.GalgameDetail, *errors.AppError) {
	wikiData, appErr := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%d", galgameID), nil)
	if appErr != nil {
		return nil, appErr
	}

	var parsed dto.WikiGalgameDetailFullResp
	if err := json.Unmarshal(wikiData, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}
	g := parsed.Galgame
	if g.Status == 1 {
		return nil, errors.ErrNotFound("该 Galgame 已被封禁")
	}

	// Async view bump (don't block the response).
	go s.galgameRepo.IncrementView(galgameID)

	local := s.galgameRepo.FindLocal(galgameID)
	isLiked, isFavorited := s.interactionRepo.UserInteraction(currentUserID, galgameID)

	platforms, languages, types := s.resourceMetaRepo.FindResourceMetaByGalgame(galgameID)

	var series *dto.GalgameDetailSeries
	if g.SeriesID != nil {
		series = s.fetchSeriesBrief(ctx, *g.SeriesID)
	}

	ratings := s.buildDetailRatings(galgameID, currentUserID, g)

	detail := galgameDetailFromWiki(g, parsed.Users)
	detail.View = local.View
	detail.LikeCount = local.LikeCount
	detail.FavoriteCount = local.FavoriteCount
	detail.IsLiked = isLiked
	detail.IsFavorited = isFavorited
	detail.Platform = platforms
	detail.Language = languages
	detail.Type = types
	detail.Series = series
	detail.Ratings = ratings
	return &detail, nil
}

// fetchSeriesBrief loads a minimal series summary (used on galgame detail page).
func (s *GalgameService) fetchSeriesBrief(ctx context.Context, seriesID int) *dto.GalgameDetailSeries {
	data, err := s.wikiClient.Get(ctx, fmt.Sprintf("/series/%d", seriesID), nil)
	if err != nil {
		return nil
	}
	var brief dto.WikiSeriesBrief
	if jsonErr := json.Unmarshal(data, &brief); jsonErr != nil {
		return nil
	}

	isNSFW := false
	samples := make([]dto.GalgameSample, 0, min(len(brief.Galgame), 5))
	for i, sg := range brief.Galgame {
		if sg.ContentLimit == "nsfw" {
			isNSFW = true
		}
		if i < 5 {
			samples = append(samples, dto.GalgameSample{
				Name: dto.KunLanguage{
					EnUs: sg.NameEnUs, JaJp: sg.NameJaJp,
					ZhCn: sg.NameZhCn, ZhTw: sg.NameZhTw,
				},
				Banner: sg.Banner,
			})
		}
	}
	return &dto.GalgameDetailSeries{
		ID:            brief.ID,
		Name:          brief.Name,
		Description:   brief.Description,
		IsNSFW:        isNSFW,
		SampleGalgame: samples,
		GalgameCount:  len(brief.Galgame),
		Created:       brief.Created,
		Updated:       brief.Updated,
	}
}

// buildDetailRatings assembles the ratings list with user resolution and liked flag.
func (s *GalgameService) buildDetailRatings(
	galgameID, currentUserID int,
	g dto.WikiGalgameDetailFull,
) []dto.GalgameDetailRating {
	rows := s.detailRatingRepo.FindRatingsByGalgame(galgameID)
	if len(rows) == 0 {
		return []dto.GalgameDetailRating{}
	}

	userIDs := make([]int, len(rows))
	ratingIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		ratingIDs[i] = r.ID
	}
	userMap := s.galgameRepo.FindUsersByIDs(userIDs)
	likedSet := s.detailRatingRepo.FindLikedRatingIDs(currentUserID, ratingIDs)

	out := make([]dto.GalgameDetailRating, len(rows))
	for i, r := range rows {
		out[i] = detailRatingFromRow(r, userMap[r.UserID], likedSet[r.ID], galgameID, g)
	}
	return out
}

// ──────────────────────────────────────────
// GetList — GET /galgame
// ──────────────────────────────────────────

func (s *GalgameService) GetList(
	ctx context.Context,
	req *dto.GalgameListRequest,
) (*dto.GalgameListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	filter := model.GalgameListFilter{
		Type:                 req.Type,
		Language:             req.Language,
		Platform:             req.Platform,
		SortField:            req.SortField,
		SortOrder:            sortOrder,
		IncludeProviders:     splitCSV(req.IncludeProviders),
		ExcludeOnlyProviders: splitCSV(req.ExcludeOnlyProviders),
		Page:                 req.Page,
		Limit:                req.Limit,
	}

	ids, total := s.listRepo.ListIDs(filter)
	if len(ids) == 0 {
		return &dto.GalgameListPage{Galgames: []dto.GalgameListCard{}, Total: total}, nil
	}

	// Wiki batch metadata
	briefMap, _ := s.wikiClient.GetBatch(ctx, ids)
	if briefMap == nil {
		briefMap = map[int]client.GalgameBrief{}
	}

	// Users (from wiki briefs)
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := s.galgameRepo.FindUsersByIDs(userIDs)

	// Local stats batch
	localMap := s.galgameRepo.FindLocalBatch(ids)

	// Platform/language aggregation
	metaRows := s.resourceMetaRepo.FindResourceMetaBatch(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	cards := make([]dto.GalgameListCard, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		cards = append(cards, dto.GalgameListCard{
			ID: id,
			Name: dto.KunLanguage{
				EnUs: b.NameEnUs, JaJp: b.NameJaJp,
				ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
			},
			Banner:             b.Banner,
			User:               userBriefToDTO(userMap[b.UserID]),
			ContentLimit:       b.ContentLimit,
			View:               localMap[id].View,
			LikeCount:          localMap[id].LikeCount,
			ResourceUpdateTime: b.ResourceUpdateTime,
			Platform:           emptyStrSliceIfNil(platformMap[id]),
			Language:           emptyStrSliceIfNil(languageMap[id]),
		})
	}
	return &dto.GalgameListPage{Galgames: cards, Total: total}, nil
}
