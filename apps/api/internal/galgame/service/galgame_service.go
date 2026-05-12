package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

// GalgameService handles the "core" galgame lifecycle: create, merge PR,
// detail aggregation, list with filters, and local interaction toggles.
type GalgameService struct {
	galgameRepo      *repository.GalgameRepository
	interactionRepo  *repository.GalgameInteractionRepository
	listRepo         *repository.GalgameListRepository
	resourceMetaRepo *repository.GalgameResourceMetaRepository
	detailRatingRepo *repository.GalgameDetailRatingRepository
	stateRepo        *userRepo.StateRepository
	wikiClient       *client.GalgameClient
	userClient       *userclient.Client
	helpers          InteractionHelpers
}

func NewGalgameService(
	galgameRepo *repository.GalgameRepository,
	interactionRepo *repository.GalgameInteractionRepository,
	listRepo *repository.GalgameListRepository,
	resourceMetaRepo *repository.GalgameResourceMetaRepository,
	detailRatingRepo *repository.GalgameDetailRatingRepository,
	stateRepo *userRepo.StateRepository,
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		interactionRepo:  interactionRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		detailRatingRepo: detailRatingRepo,
		stateRepo:        stateRepo,
		wikiClient:       wikiClient,
		userClient:       userClient,
	}
}

// ──────────────────────────────────────────
// Create — POST /galgame
// ──────────────────────────────────────────

// Create forwards the payload to wiki, then awards moemoepoint and creates
// the local stub row for the new galgame. Returns the raw wiki response body
// so the handler can forward it verbatim.
//
// Daily-limit policy (mirrors topic create, formerly nitro
// api/galgame/index.post.ts:43): a user can create at most
// `moemoepoint/10 + 1` galgames per 24h. The limit is checked BEFORE the
// wiki call so we don't pollute wiki with rejects, using wiki's own
// `galgame_created_today` stat as the canonical day-count (kungal has no
// local creation log post-migration). There's still a thin race window
// between the check and wiki accepting the create — acceptable because
// wiki rejects duplicate vndb_id, which is the main spam vector.
//
// Post-success local side effects (stub row + moemoepoint +3) run inside
// a single transaction with SELECT … FOR UPDATE on kungal_user_state, so
// concurrent self-double-submits can't double-reward.
func (s *GalgameService) Create(
	ctx context.Context,
	userID int,
	token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	// Daily-limit gate.
	state, err := s.stateRepo.FindByID(userID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}
	dailyLimit := int64(state.Moemoepoint/10 + 1)
	if wikiStats, sErr := s.wikiClient.GetUserStats(ctx, userID); sErr == nil && wikiStats != nil {
		if wikiStats.GalgameCreatedToday >= dailyLimit {
			return nil, errors.ErrBadRequest("您今日发布的 Galgame 已达上限")
		}
	}
	// On wiki stats failure we choose to allow the create rather than
	// hard-fail — wiki itself remains the authority on VNDB-ID uniqueness.

	// Forward to wiki.
	data, appErr := s.wikiClient.PostWithToken(ctx, "/galgame", token, json.RawMessage(body), contentType)
	if appErr != nil {
		return nil, appErr
	}

	var created dto.WikiCreatedResp
	_ = json.Unmarshal(data, &created)

	if created.ID > 0 {
		txErr := s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			// Lock the kungal_user_state row so a concurrent create on the
			// same account can't both pass the check above AND both award +3.
			if _, lockErr := s.stateRepo.LockForUpdate(tx, userID); lockErr != nil {
				return lockErr
			}
			s.galgameRepo.CreateLocalStub(tx, created.ID)
			if pErr := s.stateRepo.AdjustMoemoepointTx(tx, userID, constants.RewardCreateGalgame); pErr != nil {
				return pErr
			}
			return nil
		})
		if txErr != nil {
			// Wiki already accepted the create; leaving the user without the
			// +3 reward is preferable to half-rolling-back wiki state.
			// Surface as a soft error in logs but return the wiki response.
			slog.Warn("galgame 创建本地副作用失败 (wiki 已成功)",
				"gid", created.ID, "uid", userID, "error", txErr)
		}
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
		ctx, fmt.Sprintf("/galgame/%s/prs/%s/merge", gid, prID), token, nil, "",
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

	ratings := s.buildDetailRatings(ctx, galgameID, currentUserID, g)

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
	ctx context.Context,
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
	userMap := s.userClient.Hydrate(ctx, userIDs)
	likedSet := s.detailRatingRepo.FindLikedRatingIDs(currentUserID, ratingIDs)

	out := make([]dto.GalgameDetailRating, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		out = append(out, detailRatingFromRow(r, u, likedSet[r.ID], galgameID, g))
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

	// Users (from wiki briefs) — hydrated from OAuth.
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

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
