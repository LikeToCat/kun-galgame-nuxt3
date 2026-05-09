package service

import (
	"context"
	"math/rand/v2"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

// UserService surfaces the kungal-side user pages: profile, status,
// floating card, and check-in. Identity (name / avatar / bio / status /
// roles) is fetched from OAuth via userclient. Per-site state
// (moemoepoint, daily counters) lives in kungal_user_state. Content
// counts (topic / reply / etc.) come from local aggregates.
type UserService struct {
	stateRepo     *repository.StateRepository
	userStatsRepo *repository.UserStatsRepository
	rdb           *redis.Client
	wikiClient    *galgameClient.GalgameClient
	userClient    *userclient.Client
}

func NewUserService(
	stateRepo *repository.StateRepository,
	userStatsRepo *repository.UserStatsRepository,
	rdb *redis.Client,
	wikiClient *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *UserService {
	return &UserService{
		stateRepo:     stateRepo,
		userStatsRepo: userStatsRepo,
		rdb:           rdb,
		wikiClient:    wikiClient,
		userClient:    userClient,
	}
}

// ──────────────────────────────────────────
// Profile
// ──────────────────────────────────────────

func (s *UserService) GetUserProfile(ctx context.Context, uid int) (*dto.UserProfileDetail, *errors.AppError) {
	u, ok, err := s.userClient.User(ctx, uid)
	if err != nil {
		return nil, errors.ErrInternal("查询用户信息失败")
	}
	if !ok {
		return nil, errors.ErrNotFound("未找到该用户")
	}
	// Banned users surface only id+name so callers can render a "已封禁"
	// placeholder; we don't leak avatar / bio / stats.
	if u.Status != 0 {
		return &dto.UserProfileDetail{ID: u.ID, Name: u.Name, Status: u.Status}, nil
	}

	stats, err := s.userStatsRepo.GetUserStats(uid)
	if err != nil {
		return nil, errors.ErrInternal("获取用户统计失败")
	}
	state, _ := s.stateRepo.FindByID(uid)
	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}

	profile := &dto.UserProfileDetail{
		ID:          u.ID,
		Name:        u.Name,
		Avatar:      u.Avatar,
		Role:        middleware.RoleFromOAuthRoles(u.Roles),
		Status:      u.Status,
		Moemoepoint: moe,
		Bio:         u.Bio,
	}
	// CreatedAt: kungal_user_state.created marks "first seen on this
	// site" — close enough for a community profile.
	if state != nil {
		profile.CreatedAt = state.CreatedAt
	}

	profile.Topic = stats.Topic
	profile.TopicPoll = stats.TopicPoll
	profile.ReplyCreated = stats.ReplyCreated
	profile.CommentCreated = stats.CommentCreated
	profile.GalgameComment = stats.GalgameComment
	profile.GalgameRating = stats.GalgameRating
	profile.GalgameResource = stats.GalgameResource
	profile.GalgameToolset = stats.GalgameToolset
	profile.GalgameToolsetResource = stats.GalgameToolsetResource
	profile.Upvote = stats.Upvote
	profile.Like = stats.Like
	profile.Dislike = stats.Dislike
	profile.DailyTopicCount = stats.DailyTopicCount

	if wikiStats, err := s.wikiClient.GetUserStats(ctx, uid); err == nil && wikiStats != nil {
		profile.Galgame = wikiStats.GalgameCreated
		profile.DailyGalgameCount = wikiStats.GalgameCreatedToday
		profile.ContributeGalgame = wikiStats.GalgameContributed
	}

	return profile, nil
}

// ──────────────────────────────────────────
// Check-in / status
// ──────────────────────────────────────────

func (s *UserService) CheckIn(ctx context.Context, uid int) (int, *errors.AppError) {
	state, err := s.stateRepo.FindByID(uid)
	if err != nil {
		return 0, errors.ErrNotFound("未找到用户")
	}
	if state.DailyCheckIn != 0 {
		return 0, errors.ErrBadRequest("您今天已经签到过了")
	}

	points := rand.IntN(8) // 0-7
	if err := s.stateRepo.CheckIn(uid, points); err != nil {
		return 0, errors.ErrInternal("签到失败")
	}
	return points, nil
}

func (s *UserService) GetUserStatus(ctx context.Context, uid int) (*dto.UserStatusResponse, *errors.AppError) {
	state, err := s.stateRepo.FindByID(uid)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}

	unreadMessage, _ := s.userStatsRepo.CountUnreadMessages(uid)
	unreadSystem, _ := s.userStatsRepo.CountUnreadSystemMessages()
	unreadChat, _ := s.userStatsRepo.CountUnreadChatMessages(uid)

	return &dto.UserStatusResponse{
		Moemoepoints:  state.Moemoepoint,
		IsCheckIn:     state.DailyCheckIn == 1,
		HasNewMessage: (unreadMessage + unreadSystem + unreadChat) > 0,
	}, nil
}

// ──────────────────────────────────────────
// Floating hover card
// ──────────────────────────────────────────

func (s *UserService) GetFloatingCard(ctx context.Context, uid int) (*dto.FloatingCardResponse, *errors.AppError) {
	u, ok, err := s.userClient.User(ctx, uid)
	if err != nil {
		return nil, errors.ErrInternal("查询用户信息失败")
	}
	if !ok || u.Status != 0 {
		// Banned or missing — kungal hides the card per agreed policy.
		return nil, errors.ErrNotFound("未找到该用户")
	}

	state, _ := s.stateRepo.FindByID(uid)
	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}

	stats := s.userStatsRepo.FindFloatingStats(uid)
	return &dto.FloatingCardResponse{
		ID:                   u.ID,
		Name:                 u.Name,
		Avatar:               u.Avatar,
		Moemoepoint:          moe,
		TopicCount:           stats.TopicCount,
		TopicReplyCount:      stats.TopicReplyCount,
		TopicCommentCount:    stats.TopicCommentCount,
		GalgameResourceCount: stats.ResourceCount,
	}, nil
}
