package service

import (
	"context"
	"encoding/json"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
)

type UserContentService struct {
	userContentRepo *repository.UserContentRepository
	userBriefRepo   *repository.UserBriefRepository
	wikiClient      *galgameClient.GalgameClient
}

func NewUserContentService(
	userContentRepo *repository.UserContentRepository,
	userBriefRepo *repository.UserBriefRepository,
	wikiClient *galgameClient.GalgameClient,
) *UserContentService {
	return &UserContentService{
		userContentRepo: userContentRepo,
		userBriefRepo:   userBriefRepo,
		wikiClient:      wikiClient,
	}
}

// ──────────────────────────────────────────
// User galgame list — GET /user/:uid/galgames
// ──────────────────────────────────────────

// GetUserGalgameCards returns enriched galgame cards for the user's list
// (created / liked / favorited / commented depending on req.Type).
func (s *UserContentService) GetUserGalgameCards(
	ctx context.Context,
	uid int,
	req *dto.UserGalgamesRequest,
) ([]dto.UserGalgameCard, int64, *errors.AppError) {
	ids, total, err := s.userContentRepo.FindUserGalgameIDs(uid, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户 Galgame 列表失败")
	}
	if len(ids) == 0 {
		return []dto.UserGalgameCard{}, total, nil
	}

	briefMap, wikiErr := s.wikiClient.GetBatch(ctx, ids)
	if wikiErr != nil {
		// Wiki failure → return empty list but preserve total count.
		return []dto.UserGalgameCard{}, total, nil
	}

	localMap := s.userContentRepo.FindGalgameLocalStats(ids)
	metaRows := s.userContentRepo.FindResourceMetaByGalgameIDs(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	userIDs := collectUniqueIDs(values(briefMap), func(b galgameClient.GalgameBrief) int { return b.UserID })
	userMap := s.userBriefRepo.FindUsersByIDs(userIDs)

	cards := make([]dto.UserGalgameCard, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		l := localMap[id]
		cards = append(cards, dto.UserGalgameCard{
			ID:                 b.ID,
			Name:               briefToLocale(b),
			Banner:             b.Banner,
			User:               userBriefRowToDTO(userMap[b.UserID]),
			ContentLimit:       b.ContentLimit,
			View:               l.View,
			LikeCount:          l.LikeCount,
			ResourceUpdateTime: b.ResourceUpdateTime,
			Platform:           emptyStrSlice(platformMap[id]),
			Language:           emptyStrSlice(languageMap[id]),
		})
	}
	return cards, total, nil
}

// values is a tiny helper that extracts map values into a slice.
func values[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// ──────────────────────────────────────────
// Topics / replies / comments (already thin)
// ──────────────────────────────────────────

func (s *UserContentService) GetUserTopics(ctx context.Context, uid int, req *dto.UserTopicsRequest) ([]dto.UserTopic, int64, *errors.AppError) {
	items, total, err := s.userContentRepo.FindUserTopics(uid, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户话题列表失败")
	}
	return items, total, nil
}

func (s *UserContentService) GetUserReplies(ctx context.Context, uid int, req *dto.UserRepliesRequest) ([]repository.UserReply, int64, *errors.AppError) {
	items, total, err := s.userContentRepo.FindUserReplies(uid, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户回复列表失败")
	}
	return items, total, nil
}

func (s *UserContentService) GetUserComments(ctx context.Context, uid int, req *dto.UserCommentsRequest) ([]repository.UserComment, int64, *errors.AppError) {
	items, total, err := s.userContentRepo.FindUserComments(uid, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户评论列表失败")
	}
	return items, total, nil
}

// ──────────────────────────────────────────
// Resources — GET /user/:uid/resources
// ──────────────────────────────────────────

func (s *UserContentService) GetUserResources(
	ctx context.Context,
	uid int,
	req *dto.UserResourcesRequest,
) (*dto.UserResourcesResponse, *errors.AppError) {
	rows, total, err := s.userContentRepo.FindUserResources(uid, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户资源列表失败")
	}

	resourceIDs := make([]int, len(rows))
	galgameIDs := collectUniqueIDs(rows, func(r repository.UserResource) int { return r.GalgameID })
	for i, r := range rows {
		resourceIDs[i] = r.ID
	}

	var linkMap map[int][]string
	if len(resourceIDs) > 0 {
		linkMap, _ = s.userContentRepo.FindResourceLinks(resourceIDs)
	}

	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = s.wikiClient.GetBatch(ctx, galgameIDs)
	}

	items := make([]dto.UserResourceItem, len(rows))
	for i, r := range rows {
		links := linkMap[r.ID]
		if links == nil {
			links = []string{}
		}
		name := emptyLocale()
		if briefMap != nil {
			if b, ok := briefMap[r.GalgameID]; ok {
				name = briefToLocale(b)
			}
		}
		items[i] = dto.UserResourceItem{
			ID:          r.ID,
			GalgameID:   r.GalgameID,
			GalgameName: name,
			Type:        r.Type,
			Language:    r.Language,
			Platform:    r.Platform,
			Size:        r.Size,
			Link:        links,
			Code:        r.Code,
			Password:    r.Password,
			Note:        r.Note,
			Status:      r.Status,
			Created:     r.Created,
		}
	}

	return &dto.UserResourcesResponse{Resources: items, Total: total}, nil
}

// ──────────────────────────────────────────
// Ratings — GET /user/:uid/ratings
// ──────────────────────────────────────────

func (s *UserContentService) GetUserRatings(
	ctx context.Context,
	uid int,
	req *dto.UserRatingsRequest,
) (*dto.UserRatingsResponse, *errors.AppError) {
	rows, total, err := s.userContentRepo.FindUserRatings(uid, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户评分列表失败")
	}

	galgameIDs := collectUniqueIDs(rows, func(r repository.UserRating) int { return r.GalgameID })
	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = s.wikiClient.GetBatch(ctx, galgameIDs)
	}

	items := make([]dto.UserRatingItem, len(rows))
	for i, r := range rows {
		var galgameType []string
		if r.GalgameType != "" {
			_ = json.Unmarshal([]byte(r.GalgameType), &galgameType)
		}

		galgame := dto.UserRatingGalgame{ID: r.GalgameID}
		if briefMap != nil {
			if b, ok := briefMap[r.GalgameID]; ok {
				galgame = dto.UserRatingGalgame{
					ID:           b.ID,
					Name:         briefToLocale(b),
					ContentLimit: b.ContentLimit,
				}
			}
		}

		items[i] = dto.UserRatingItem{
			ID:           r.ID,
			User:         dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Recommend:    r.Recommend,
			Overall:      r.Overall,
			View:         r.View,
			GalgameType:  galgameType,
			PlayStatus:   r.PlayStatus,
			Art:          r.Art,
			Story:        r.Story,
			Music:        r.Music,
			Character:    r.Character,
			Route:        r.Route,
			System:       r.System,
			Voice:        r.Voice,
			ReplayValue:  r.ReplayValue,
			SpoilerLevel: r.SpoilerLevel,
			LikeCount:    r.LikeCount,
			Created:      r.Created,
			Updated:      r.Updated,
			Galgame:      galgame,
		}
	}

	return &dto.UserRatingsResponse{RatingData: items, Total: total}, nil
}
