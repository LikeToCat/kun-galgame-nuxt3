package service

import (
	"context"
	"math/rand/v2"
	"strings"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	userRepo      *repository.UserRepository
	userStatsRepo *repository.UserStatsRepository
	userBriefRepo *repository.UserBriefRepository
	rdb           *redis.Client
	wikiClient    *galgameClient.GalgameClient
}

func NewUserService(
	userRepo *repository.UserRepository,
	userStatsRepo *repository.UserStatsRepository,
	userBriefRepo *repository.UserBriefRepository,
	rdb *redis.Client,
	wikiClient *galgameClient.GalgameClient,
) *UserService {
	return &UserService{
		userRepo:      userRepo,
		userStatsRepo: userStatsRepo,
		userBriefRepo: userBriefRepo,
		rdb:           rdb,
		wikiClient:    wikiClient,
	}
}

// ──────────────────────────────────────────
// Profile / account
// ──────────────────────────────────────────

func (s *UserService) GetUserProfile(ctx context.Context, uid int) (*dto.UserProfileDetail, *errors.AppError) {
	user, err := s.userRepo.FindByID(uid)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}
	if user.Status == 1 {
		return &dto.UserProfileDetail{
			ID:     user.ID,
			Name:   user.Name,
			Status: 1,
		}, nil
	}

	stats, err := s.userStatsRepo.GetUserStats(uid)
	if err != nil {
		return nil, errors.ErrInternal("获取用户统计失败")
	}

	profile := &dto.UserProfileDetail{
		ID:          user.ID,
		Name:        user.Name,
		Avatar:      user.Avatar,
		Role:        user.Role,
		Status:      user.Status,
		Moemoepoint: user.Moemoepoint,
		Bio:         user.Bio,
		CreatedAt:   user.CreatedAt,

		Topic:                  stats.Topic,
		TopicPoll:              stats.TopicPoll,
		ReplyCreated:           stats.ReplyCreated,
		CommentCreated:         stats.CommentCreated,
		GalgameComment:         stats.GalgameComment,
		GalgameRating:          stats.GalgameRating,
		GalgameResource:        stats.GalgameResource,
		GalgameToolset:         stats.GalgameToolset,
		GalgameToolsetResource: stats.GalgameToolsetResource,

		Upvote:  stats.Upvote,
		Like:    stats.Like,
		Dislike: stats.Dislike,

		DailyTopicCount: stats.DailyTopicCount,
	}

	// Enrich with wiki galgame stats (non-blocking — zero values on failure)
	if wikiStats, err := s.wikiClient.GetUserStats(ctx, uid); err == nil && wikiStats != nil {
		profile.Galgame = wikiStats.GalgameCreated
		profile.DailyGalgameCount = wikiStats.GalgameCreatedToday
		profile.ContributeGalgame = wikiStats.GalgameContributed
	}

	return profile, nil
}

func (s *UserService) CheckIn(ctx context.Context, uid int) (int, *errors.AppError) {
	user, err := s.userRepo.FindByID(uid)
	if err != nil {
		return 0, errors.ErrNotFound("未找到用户")
	}
	if user.DailyCheckIn != 0 {
		return 0, errors.ErrBadRequest("您今天已经签到过了")
	}

	points := rand.IntN(8) // 0-7
	if err := s.userRepo.CheckIn(uid, points); err != nil {
		return 0, errors.ErrInternal("签到失败")
	}

	return points, nil
}

func (s *UserService) UpdateBio(ctx context.Context, uid int, bio string) *errors.AppError {
	if err := s.userRepo.UpdateField(uid, "bio", bio); err != nil {
		return errors.ErrInternal("更新签名失败")
	}
	return nil
}

func (s *UserService) UpdateUsername(ctx context.Context, uid int, username string) *errors.AppError {
	user, err := s.userRepo.FindByID(uid)
	if err != nil {
		return errors.ErrNotFound("未找到该用户")
	}
	if user.Moemoepoint < 17 {
		return errors.ErrBadRequest("更改用户名需要 17 萌萌点, 您的萌萌点不足")
	}

	exists, err := s.userRepo.UsernameExists(username)
	if err != nil {
		return errors.ErrInternal("查询用户名失败")
	}
	if exists {
		return errors.ErrBadRequest("您的用户名已经被使用, 请换一个")
	}

	if err := s.userRepo.UpdateUsernameWithCost(uid, username, 17); err != nil {
		return errors.ErrInternal("更新用户名失败")
	}
	return nil
}

func (s *UserService) UpdateEmail(ctx context.Context, uid int, req *dto.UpdateEmailRequest) *errors.AppError {
	codeKey := req.CodeSalt + ":" + req.Email
	valid, err := s.verifyCode(ctx, codeKey, req.Code)
	if err != nil || !valid {
		return errors.ErrBadRequest("错误的邮箱验证码")
	}
	s.rdb.Del(ctx, codeKey)

	if err := s.userRepo.UpdateField(uid, "email", req.Email); err != nil {
		return errors.ErrInternal("更新邮箱失败")
	}
	return nil
}

func (s *UserService) GetMaskedEmail(ctx context.Context, uid int) (string, *errors.AppError) {
	user, err := s.userRepo.FindByID(uid)
	if err != nil {
		return "", errors.ErrNotFound("未找到该用户")
	}

	email := user.Email
	atIdx := strings.IndexByte(email, '@')
	if atIdx < 0 {
		return email, nil
	}
	localPart := email[:atIdx]
	domain := email[atIdx:]

	masked := localPart
	if len(localPart) > 2 {
		masked = localPart[:2] + "~~~~~~~"
	}
	return masked + domain, nil
}

func (s *UserService) GetUserStatus(ctx context.Context, uid int) (*dto.UserStatusResponse, *errors.AppError) {
	user, err := s.userRepo.FindByID(uid)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}

	unreadMessage, _ := s.userStatsRepo.CountUnreadMessages(uid)
	unreadSystem, _ := s.userStatsRepo.CountUnreadSystemMessages()
	unreadChat, _ := s.userStatsRepo.CountUnreadChatMessages(uid)

	return &dto.UserStatusResponse{
		Moemoepoints:  user.Moemoepoint,
		IsCheckIn:     user.DailyCheckIn == 1,
		HasNewMessage: (unreadMessage + unreadSystem + unreadChat) > 0,
	}, nil
}

func (s *UserService) UploadAvatar(ctx context.Context, uid int, avatarData []byte) (string, *errors.AppError) {
	// TODO: implement S3 upload + sharp resize
	// For now return error
	return "", errors.ErrBadRequest("头像上传功能正在迁移中")
}

// ──────────────────────────────────────────
// Floating hover card
// ──────────────────────────────────────────

func (s *UserService) GetFloatingCard(uid int) (*dto.FloatingCardResponse, *errors.AppError) {
	user, err := s.userBriefRepo.FindFloatingUser(uid)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}
	if user.Status == 1 {
		return nil, errors.ErrNotFound("该用户已被封禁")
	}

	stats := s.userStatsRepo.FindFloatingStats(uid)
	return &dto.FloatingCardResponse{
		ID:                   user.ID,
		Name:                 user.Name,
		Avatar:               user.Avatar,
		Moemoepoint:          user.Moemoepoint,
		TopicCount:           stats.TopicCount,
		TopicReplyCount:      stats.TopicReplyCount,
		TopicCommentCount:    stats.TopicCommentCount,
		GalgameResourceCount: stats.ResourceCount,
	}, nil
}

// ──────────────────────────────────────────
// Admin / shared
// ──────────────────────────────────────────

func (s *UserService) verifyCode(ctx context.Context, key, code string) (bool, error) {
	stored, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return stored == code, nil
}

// BanUser bans or unbans a user (admin only).
func (s *UserService) BanUser(ctx context.Context, uid int, status int) *errors.AppError {
	if err := s.userRepo.UpdateField(uid, "status", status); err != nil {
		return errors.ErrInternal("更新用户状态失败")
	}
	return nil
}

// DeleteUser permanently deletes a user (admin only).
func (s *UserService) DeleteUser(ctx context.Context, uid int) *errors.AppError {
	if err := s.userRepo.Delete(uid); err != nil {
		return errors.ErrInternal("删除用户失败")
	}
	return nil
}
