package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type RatingService struct {
	ratingRepo *repository.RatingRepository
	wikiClient *client.GalgameClient
	userClient *userclient.Client
	helpers    InteractionHelpers
}

func NewRatingService(
	ratingRepo *repository.RatingRepository,
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
) *RatingService {
	return &RatingService{ratingRepo: ratingRepo, wikiClient: wikiClient, userClient: userClient}
}

// ratingReward applies the documented thresholds (architecture-patterns spec):
// summary < 233 → +3, < 666 → +5, ≥ 666 → +10.
func ratingReward(summaryLen int) int {
	switch {
	case summaryLen >= constants.RatingLenThresholdHigh:
		return constants.RatingRewardHigh
	case summaryLen >= constants.RatingLenThresholdMedium:
		return constants.RatingRewardMedium
	default:
		return constants.RatingRewardLow
	}
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
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchWikiBriefs(ctx, galgameIDs)

	cards := make([]dto.RatingCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		cards = append(cards, ratingRowToCard(r, u, briefMap[r.GalgameID]))
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
	isLiked := containsInt(likerIDs, currentUserID)

	// Author + comments — collect all uids and hydrate in one batch.
	commentRows := s.ratingRepo.FindComments(ratingID)
	uidSet := map[int]struct{}{row.UserID: {}}
	for _, id := range likerIDs {
		uidSet[id] = struct{}{}
	}
	for _, cm := range commentRows {
		uidSet[cm.UserID] = struct{}{}
		if cm.TargetUserID != nil {
			uidSet[*cm.TargetUserID] = struct{}{}
		}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		uids = append(uids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	comments := make([]dto.RatingCommentItem, len(commentRows))
	for i, cm := range commentRows {
		comments[i] = ratingCommentRowToDTO(cm, userMap)
	}

	// Galgame detail from wiki
	galgame := s.buildRatingGalgame(ctx, row.GalgameID)

	// Author projection of liked users (preserve liker order).
	authorBriefs := make([]dto.UserBrief, 0, len(likerIDs))
	for _, id := range likerIDs {
		u := userMap[id]
		if !userclient.IsRenderable(u) {
			continue
		}
		authorBriefs = append(authorBriefs, userBriefToDTO(u))
	}

	detail := &dto.RatingDetail{
		ID:           row.ID,
		User:         userBriefToDTO(userMap[row.UserID]),
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
// CreateRating — POST /galgame-rating
// One rating per user per galgame. Reward tier follows summary length.
// ──────────────────────────────────────────

func (s *RatingService) CreateRating(
	ctx context.Context,
	uid int,
	req *dto.CreateRatingRequest,
) (*dto.CreatedRating, *errors.AppError) {
	if s.ratingRepo.ExistsByUserGalgame(req.GalgameID, uid) {
		return nil, errors.ErrBadRequest("您已经发布过该 Galgame 的评分了")
	}

	galgameTypeJSON, _ := json.Marshal(req.GalgameType)
	rating := &model.GalgameRating{
		Recommend:    req.Recommend,
		Overall:      req.Overall,
		GalgameType:  galgameTypeJSON,
		PlayStatus:   req.PlayStatus,
		ShortSummary: req.ShortSummary,
		SpoilerLevel: req.SpoilerLevel,
		Art:          req.Art, Story: req.Story, Music: req.Music,
		Character: req.Character, Route: req.Route, System: req.System,
		Voice: req.Voice, ReplayValue: req.ReplayValue,
		GalgameID: req.GalgameID, UserID: uid,
	}
	reward := ratingReward(len(req.ShortSummary))

	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.Create(tx, rating); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, uid, reward)
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建评分失败")
	}

	user, _, _ := s.userClient.User(ctx, uid)
	briefMap := s.fetchWikiBriefs(ctx, []int{req.GalgameID})

	return &dto.CreatedRating{
		ID:           rating.ID,
		User:         userBriefToDTO(user),
		Recommend:    rating.Recommend,
		Overall:      rating.Overall,
		View:         rating.View,
		GalgameType:  rawJSON(string(rating.GalgameType)),
		PlayStatus:   rating.PlayStatus,
		ShortSummary: rating.ShortSummary,
		SpoilerLevel: rating.SpoilerLevel,
		RatingScores: dto.RatingScores{
			Art: rating.Art, Story: rating.Story, Music: rating.Music,
			Character: rating.Character, Route: rating.Route, System: rating.System,
			Voice: rating.Voice, ReplayValue: rating.ReplayValue,
		},
		LikeCount: 0, IsLiked: false,
		Created: rating.CreatedAt.Format(time.RFC3339),
		Updated: rating.UpdatedAt.Format(time.RFC3339),
		Galgame: dto.RatingGalgameBrief{
			ID:           req.GalgameID,
			ContentLimit: briefMap[req.GalgameID].ContentLimit,
			Name: dto.KunLanguage{
				EnUs: briefMap[req.GalgameID].NameEnUs,
				JaJp: briefMap[req.GalgameID].NameJaJp,
				ZhCn: briefMap[req.GalgameID].NameZhCn,
				ZhTw: briefMap[req.GalgameID].NameZhTw,
			},
		},
	}, nil
}

// ──────────────────────────────────────────
// UpdateRating — PUT /galgame-rating/:id
// Author-only. Adjusts moemoepoint by the reward-tier delta when the summary
// length crosses a threshold.
// ──────────────────────────────────────────

func (s *RatingService) UpdateRating(
	uid int,
	req *dto.UpdateRatingRequest,
) *errors.AppError {
	rating, err := s.ratingRepo.FindRatingForWrite(req.GalgameRatingID)
	if err != nil {
		return errors.ErrNotFound("评分不存在")
	}
	if rating.UserID != uid {
		return errors.ErrForbidden("您无权限修改他人评分")
	}

	pointDiff := ratingReward(len(req.ShortSummary)) - ratingReward(len(rating.ShortSummary))
	galgameTypeJSON, _ := json.Marshal(req.GalgameType)
	fields := map[string]any{
		"recommend":     req.Recommend,
		"overall":       req.Overall,
		"galgame_type":  galgameTypeJSON,
		"play_status":   req.PlayStatus,
		"short_summary": req.ShortSummary,
		"spoiler_level": req.SpoilerLevel,
		"art":           req.Art, "story": req.Story, "music": req.Music,
		"character": req.Character, "route": req.Route, "system": req.System,
		"voice": req.Voice, "replay_value": req.ReplayValue,
	}

	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.Update(tx, req.GalgameRatingID, fields); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, uid, pointDiff)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新评分失败")
	}
	return nil
}

// ──────────────────────────────────────────
// DeleteRating — DELETE /galgame-rating/:id
// Allowed by author, by galgame owner, or admin. Refunds the original reward.
// ──────────────────────────────────────────

func (s *RatingService) DeleteRating(
	uid, role, ratingID int,
) *errors.AppError {
	rating, err := s.ratingRepo.FindRatingForWrite(ratingID)
	if err != nil {
		return errors.ErrNotFound("未找到评分")
	}
	galgameOwner := s.ratingRepo.FindGalgameOwner(rating.GalgameID)
	if rating.UserID != uid && galgameOwner != uid && role < 2 {
		return errors.ErrForbidden("没有删除该评分的权限")
	}

	refund := -ratingReward(len(rating.ShortSummary))
	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.DeleteByID(tx, ratingID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, rating.UserID, refund)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除评分失败")
	}
	return nil
}

// ──────────────────────────────────────────
// ToggleRatingLike — PUT /galgame-rating/:id/like
// Self-like is rejected. Like grants +1 to the rating author; unlike reverses.
// Sends a deduped 'liked' notification with the summary preview as content.
// ──────────────────────────────────────────

func (s *RatingService) ToggleRatingLike(
	uid int,
	req *dto.ToggleRatingLikeRequest,
) *errors.AppError {
	rating, err := s.ratingRepo.FindRatingForWrite(req.GalgameRatingID)
	if err != nil {
		return errors.ErrNotFound("评分不存在")
	}
	if rating.UserID == uid {
		return errors.ErrBadRequest("不能给自己的评分点赞")
	}

	preview := truncate(rating.ShortSummary, constants.TextPreviewLength)
	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, has := s.ratingRepo.FindLike(tx, req.GalgameRatingID, uid)
		var delta int
		if has {
			if err := s.ratingRepo.DeleteLike(tx, existing); err != nil {
				return err
			}
			delta = -1
		} else {
			if err := s.ratingRepo.CreateLike(tx, req.GalgameRatingID, uid); err != nil {
				return err
			}
			delta = 1
		}
		if err := s.ratingRepo.AdjustLikeCount(tx, req.GalgameRatingID, delta); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, rating.UserID, delta)
		s.helpers.CreateGalgameMessageWithContent(
			tx, uid, rating.UserID, "liked", preview, rating.GalgameID,
		)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// ──────────────────────────────────────────
// CreateRatingComment — POST /galgame-rating/:id/comment
// Sends a 'commented' notification to targetUser unless self-reply.
// ──────────────────────────────────────────

func (s *RatingService) CreateRatingComment(
	ctx context.Context,
	uid int,
	req *dto.CreateRatingCommentRequest,
) (*dto.CreatedRatingComment, *errors.AppError) {
	galgameID := s.ratingRepo.FindRatingGalgameID(req.GalgameRatingID)
	if galgameID == 0 {
		return nil, errors.ErrNotFound("未找到这个评分")
	}

	target := req.TargetUserID
	c := &model.GalgameRatingComment{
		GalgameRatingID: req.GalgameRatingID,
		UserID:          uid,
		TargetUserID:    &target,
		Content:         req.Content,
	}

	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.CreateComment(tx, c); err != nil {
			return err
		}
		if uid != target {
			s.helpers.CreateGalgameMessageWithContent(
				tx, uid, target, "commented",
				truncate(req.Content, constants.TextPreviewLength),
				galgameID,
			)
		}
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("评论失败")
	}

	user, _, _ := s.userClient.User(ctx, uid)
	targetUser, hasTarget, _ := s.userClient.User(ctx, target)
	resp := &dto.CreatedRatingComment{
		ID:      c.ID,
		Content: c.Content,
		User:    userBriefToDTO(user),
		Created: c.CreatedAt.Format(time.RFC3339),
		Updated: c.UpdatedAt.Format(time.RFC3339),
	}
	if hasTarget {
		t := userBriefToDTO(targetUser)
		resp.TargetUser = &t
	}
	return resp, nil
}

// ──────────────────────────────────────────
// UpdateRatingComment — PUT /galgame-rating/:id/comment
// Author-only. Returns the updated comment.
// ──────────────────────────────────────────

func (s *RatingService) UpdateRatingComment(
	ctx context.Context,
	uid int,
	req *dto.UpdateRatingCommentRequest,
) (*dto.CreatedRatingComment, *errors.AppError) {
	c, err := s.ratingRepo.FindCommentByID(req.GalgameRatingCommentID)
	if err != nil {
		return nil, errors.ErrNotFound("评论不存在")
	}
	if c.UserID != uid {
		return nil, errors.ErrForbidden("您无权限编辑该评论")
	}
	if err := s.ratingRepo.UpdateCommentContent(s.ratingRepo.DB(), c.ID, req.Content); err != nil {
		return nil, errors.ErrInternal("更新评论失败")
	}

	user, _, _ := s.userClient.User(ctx, c.UserID)
	resp := &dto.CreatedRatingComment{
		ID: c.ID, Content: req.Content,
		User:    userBriefToDTO(user),
		Created: c.CreatedAt.Format(time.RFC3339),
		Updated: time.Now().Format(time.RFC3339),
	}
	if c.TargetUserID != nil {
		if t, ok, _ := s.userClient.User(ctx, *c.TargetUserID); ok {
			td := userBriefToDTO(t)
			resp.TargetUser = &td
		}
	}
	return resp, nil
}

// ──────────────────────────────────────────
// DeleteRatingComment — DELETE /galgame-rating/:id/comment
// Allowed by comment author, the rated galgame owner, or admin.
// ──────────────────────────────────────────

func (s *RatingService) DeleteRatingComment(
	uid, role, commentID int,
) *errors.AppError {
	c, err := s.ratingRepo.FindCommentByID(commentID)
	if err != nil {
		return errors.ErrNotFound("评论不存在")
	}
	galgameID := s.ratingRepo.FindRatingGalgameID(c.GalgameRatingID)
	galgameOwner := s.ratingRepo.FindGalgameOwner(galgameID)
	if c.UserID != uid && galgameOwner != uid && role < 2 {
		return errors.ErrForbidden("您没有删除该评论的权限")
	}
	if err := s.ratingRepo.DeleteCommentByID(s.ratingRepo.DB(), commentID); err != nil {
		return errors.ErrInternal("删除评论失败")
	}
	return nil
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
