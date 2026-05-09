package service

import (
	"context"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type TopicService struct {
	topicRepo    *repository.TopicRepository
	listRepo     *repository.TopicListRepository
	taxonomyRepo *repository.TopicTaxonomyRepository
	rdb          *redis.Client
	userClient   *userclient.Client
	stateRepo    *userRepo.StateRepository
}

func NewTopicService(
	topicRepo *repository.TopicRepository,
	listRepo *repository.TopicListRepository,
	taxonomyRepo *repository.TopicTaxonomyRepository,
	rdb *redis.Client,
	userClient *userclient.Client,
	stateRepo *userRepo.StateRepository,
) *TopicService {
	return &TopicService{
		topicRepo:    topicRepo,
		listRepo:     listRepo,
		taxonomyRepo: taxonomyRepo,
		rdb:          rdb,
		userClient:   userClient,
		stateRepo:    stateRepo,
	}
}

// ──────────────────────────────────────────
// List
// ──────────────────────────────────────────

func (s *TopicService) GetList(
	ctx context.Context,
	req *dto.ListTopicsRequest,
	isNSFW bool,
) ([]dto.TopicCard, int64, *errors.AppError) {
	rows, total, err := s.listRepo.FindList(
		req.Page, req.Limit,
		req.SortField, req.SortOrder, req.Category,
		isNSFW,
	)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取话题列表失败")
	}

	return s.mapListRows(ctx, rows, total)
}

func (s *TopicService) GetResourceList(
	ctx context.Context,
	req *dto.ListTopicsRequest,
	isNSFW bool,
) ([]dto.TopicCard, int64, *errors.AppError) {
	rows, total, err := s.listRepo.FindResourceList(
		req.Page, req.Limit,
		req.SortField, req.SortOrder, req.Category,
		isNSFW,
	)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取资源话题列表失败")
	}

	return s.mapListRows(ctx, rows, total)
}

// mapListRows enriches topic card rows with tags+sections and maps to DTOs.
// Identity (UserName/UserAvatar) is hydrated from OAuth via userclient since
// kungal no longer keeps a local users table; banned authors get a
// placeholder so the row still renders (per agreed "hide" policy at the
// column level, full row-drop is left to the frontend).
func (s *TopicService) mapListRows(ctx context.Context, rows []repository.TopicCardRow, total int64) ([]dto.TopicCard, int64, *errors.AppError) {
	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	tagMap, _ := s.taxonomyRepo.FindTagNamesByTopicIDs(topicIDs)
	sectionMap, _ := s.taxonomyRepo.FindSectionNamesByTopicIDs(topicIDs)

	uids := userclient.CollectIDs(rows, func(r repository.TopicCardRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)
	for i := range rows {
		u := userMap[rows[i].UserID]
		rows[i].UserName = u.Name
		rows[i].UserAvatar = u.Avatar
	}

	cards := make([]dto.TopicCard, 0, len(rows))
	for i, r := range rows {
		// Drop banned authors' content from the listing.
		if u, ok := userMap[r.UserID]; ok && !userclient.IsRenderable(u) {
			continue
		}
		cards = append(cards, toTopicCard(r, tagMap[r.ID], sectionMap[r.ID], false))
		_ = i
	}
	return cards, total, nil
}

// ──────────────────────────────────────────
// Detail
// ──────────────────────────────────────────

func (s *TopicService) GetDetail(
	ctx context.Context,
	topicID int,
	userInfo *middleware.UserInfo,
) (*dto.TopicDetail, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该话题")
	}

	g, _ := errgroup.WithContext(ctx)

	var author *repository.TopicAuthorUser
	var tags []string
	var sections []string
	var hasPoll bool
	var isLiked, isDisliked, isFavorited, isUpvoted bool

	g.Go(func() error {
		// Identity from OAuth, moemoepoint from kungal_user_state.
		u, _, e := s.userClient.User(ctx, topic.UserID)
		if e != nil {
			return e
		}
		moe := 0
		if state, _ := s.stateRepo.FindByID(topic.UserID); state != nil {
			moe = state.Moemoepoint
		}
		author = &repository.TopicAuthorUser{
			ID: u.ID, Name: u.Name, Avatar: u.Avatar, Moemoepoint: moe,
		}
		return nil
	})
	g.Go(func() error {
		var e error
		tags, e = s.taxonomyRepo.FindTagNamesByTopicID(topicID)
		return e
	})
	g.Go(func() error {
		var e error
		sections, e = s.taxonomyRepo.FindSectionNamesByTopicID(topicID)
		return e
	})
	g.Go(func() error {
		var e error
		hasPoll, e = s.topicRepo.HasPoll(topicID)
		return e
	})

	if userInfo != nil {
		uid := userInfo.UID
		g.Go(func() error {
			isLiked, _ = s.topicRepo.HasUserLiked(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isDisliked, _ = s.topicRepo.HasUserDisliked(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isFavorited, _ = s.topicRepo.HasUserFavorited(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isUpvoted, _ = s.topicRepo.HasUserUpvoted(uid, topicID)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, errors.ErrInternal("获取话题详情失败")
	}

	// Increment view asynchronously
	go s.topicRepo.IncrementView(topicID)

	if tags == nil {
		tags = []string{}
	}
	if sections == nil {
		sections = []string{}
	}

	detail := &dto.TopicDetail{
		ID:          topic.ID,
		Title:       topic.Title,
		Content:     topic.Content,
		ContentHtml: markdown.Render(topic.Content),
		View:        topic.View,
		Status:      topic.Status,
		IsNSFW:      topic.IsNSFW,
		Category:    topic.Category,
		Sections:    sections,
		Tags:        tags,
		User: dto.KunUserWithMoemoepoint{
			ID:          author.ID,
			Name:        author.Name,
			Avatar:      author.Avatar,
			Moemoepoint: author.Moemoepoint,
		},
		LikeCount:        topic.LikeCount,
		IsLiked:          isLiked,
		DislikeCount:     topic.DislikeCount,
		IsDisliked:       isDisliked,
		FavoriteCount:    topic.FavoriteCount,
		IsFavorited:      isFavorited,
		UpvoteCount:      topic.UpvoteCount,
		IsUpvoted:        isUpvoted,
		ReplyCount:       topic.ReplyCount,
		IsPollTopic:      hasPoll,
		StatusUpdateTime: topic.StatusUpdateTime,
		UpvoteTime:       topic.UpvoteTime,
		Edited:           topic.Edited,
		Created:          topic.CreatedAt,
	}

	return detail, nil
}
