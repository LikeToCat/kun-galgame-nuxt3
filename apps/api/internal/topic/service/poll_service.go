package service

import (
	"context"
	"fmt"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PollService struct {
	pollRepo  *repository.PollRepository
	topicRepo *repository.TopicRepository
	rdb       *redis.Client
}

func NewPollService(
	pollRepo *repository.PollRepository,
	topicRepo *repository.TopicRepository,
	rdb *redis.Client,
) *PollService {
	return &PollService{pollRepo: pollRepo, topicRepo: topicRepo, rdb: rdb}
}

// ──────────────────────────────────────────
// Create poll
// ──────────────────────────────────────────

func (s *PollService) CreatePoll(
	ctx context.Context,
	uid, role int,
	req *dto.CreatePollRequest,
) *errors.AppError {
	topic, err := s.topicRepo.FindByID(req.TopicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限创建投票")
	}

	count, _ := s.pollRepo.CountByTopicID(req.TopicID)
	if count >= constants.MaxPollsPerTopic {
		return errors.ErrBadRequest("该话题的投票数已达上限")
	}

	var deadline *time.Time
	if req.Deadline != nil {
		if t, err := time.Parse(time.RFC3339, *req.Deadline); err == nil {
			deadline = &t
		}
	}

	txErr := s.pollRepo.DB().Transaction(func(tx *gorm.DB) error {
		poll := &topicModel.TopicPoll{
			Title:            req.Title,
			Description:      req.Description,
			Type:             req.Type,
			MinChoice:        req.MinChoice,
			MaxChoice:        req.MaxChoice,
			Deadline:         deadline,
			ResultVisibility: req.ResultVisibility,
			IsAnonymous:      req.IsAnonymous,
			CanChangeVote:    req.CanChangeVote,
			TopicID:          req.TopicID,
			UserID:           uid,
		}
		if err := s.pollRepo.CreatePoll(tx, poll); err != nil {
			return err
		}

		for _, opt := range req.Options {
			if err := s.pollRepo.CreatePollOption(tx, &topicModel.TopicPollOption{
				Text:   opt.Text,
				PollID: poll.ID,
			}); err != nil {
				return err
			}
		}

		return s.pollRepo.TouchTopicStatusUpdateTime(tx, req.TopicID, time.Now())
	})

	if txErr != nil {
		return errors.ErrInternal("创建投票失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Get polls by topic
// ──────────────────────────────────────────

func (s *PollService) GetPollsByTopic(
	ctx context.Context,
	topicID int,
	userInfo *middleware.UserInfo,
) ([]dto.TopicPollResponse, *errors.AppError) {
	polls, err := s.pollRepo.FindByTopicID(topicID)
	if err != nil {
		return nil, errors.ErrInternal("获取投票失败")
	}

	uid, role := 0, 0
	if userInfo != nil {
		uid = userInfo.UID
		role = userInfo.Role
	}

	responses := make([]dto.TopicPollResponse, 0, len(polls))
	for _, poll := range polls {
		responses = append(responses, s.buildPollResponse(&poll, uid, role))
	}
	return responses, nil
}

// ──────────────────────────────────────────
// Vote
// ──────────────────────────────────────────

func (s *PollService) Vote(
	ctx context.Context,
	uid int,
	req *dto.VoteRequest,
) *errors.AppError {
	poll, err := s.pollRepo.FindByID(req.PollID)
	if err != nil {
		return errors.ErrNotFound("未找到该投票")
	}
	if poll.Status == "closed" {
		return errors.ErrBadRequest("投票已被关闭")
	}
	if poll.Deadline != nil && time.Now().After(*poll.Deadline) {
		return errors.ErrBadRequest("投票已过截止日期")
	}

	// Validate choice count
	if poll.Type == "single" && len(req.OptionIDArray) != 1 {
		return errors.ErrBadRequest("单选投票只能选择一个选项")
	}
	if poll.Type == "multiple" {
		if len(req.OptionIDArray) < poll.MinChoice {
			return errors.ErrBadRequest("至少选择 " + fmt.Sprintf("%d", poll.MinChoice) + " 个选项")
		}
		if len(req.OptionIDArray) > poll.MaxChoice {
			return errors.ErrBadRequest("最多选择 " + fmt.Sprintf("%d", poll.MaxChoice) + " 个选项")
		}
	}

	hasVoted, _ := s.pollRepo.HasUserVoted(req.PollID, uid)
	if hasVoted && !poll.CanChangeVote {
		return errors.ErrBadRequest("该投票不允许修改投票结果")
	}

	txErr := s.pollRepo.DB().Transaction(func(tx *gorm.DB) error {
		if hasVoted {
			oldOptionIDs, _ := s.pollRepo.FindUserVoteOptionIDs(req.PollID, uid)
			if err := s.pollRepo.DeleteUserVotes(tx, req.PollID, uid); err != nil {
				return err
			}
			for _, oid := range oldOptionIDs {
				if err := s.pollRepo.AdjustOptionVoteCount(tx, oid, -1); err != nil {
					return err
				}
			}
		}

		for _, optionID := range req.OptionIDArray {
			if err := s.pollRepo.CreateVote(tx, req.PollID, optionID, uid); err != nil {
				return err
			}
			if err := s.pollRepo.AdjustOptionVoteCount(tx, optionID, 1); err != nil {
				return err
			}
		}
		return nil
	})

	if txErr != nil {
		return errors.ErrInternal("投票失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete poll
// ──────────────────────────────────────────

func (s *PollService) DeletePoll(
	ctx context.Context,
	uid, role, pollID int,
) *errors.AppError {
	poll, err := s.pollRepo.FindByID(pollID)
	if err != nil {
		return errors.ErrNotFound("未找到该投票")
	}
	if poll.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此投票")
	}

	txErr := s.pollRepo.DB().Transaction(func(tx *gorm.DB) error {
		return s.pollRepo.DeletePollCascade(tx, pollID)
	})

	if txErr != nil {
		return errors.ErrInternal("删除投票失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Vote log
// ──────────────────────────────────────────

func (s *PollService) GetVoteLog(
	ctx context.Context,
	pollID, page, limit int,
	userInfo *middleware.UserInfo,
) ([]dto.PollVoteLogEntry, int64, *errors.AppError) {
	poll, err := s.pollRepo.FindByID(pollID)
	if err != nil {
		return nil, 0, errors.ErrNotFound("未找到该投票")
	}

	uid, role := 0, 0
	if userInfo != nil {
		uid = userInfo.UID
		role = userInfo.Role
	}

	hasVoted, _ := s.pollRepo.HasUserVoted(pollID, uid)
	if !canViewResults(poll, uid, role, hasVoted) || poll.IsAnonymous {
		return []dto.PollVoteLogEntry{}, 0, nil
	}

	rows, total, err := s.pollRepo.FindVoteLogs(pollID, page, limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取投票记录失败")
	}

	entries := make([]dto.PollVoteLogEntry, len(rows))
	for i, r := range rows {
		entries[i] = dto.PollVoteLogEntry{
			ID:     r.ID,
			User:   dto.KunUser{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Option: r.OptionText,
		}
	}
	return entries, total, nil
}
