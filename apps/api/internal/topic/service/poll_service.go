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
		t, err := time.Parse(time.RFC3339, *req.Deadline)
		if err == nil {
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
		if err := tx.Create(poll).Error; err != nil {
			return err
		}

		for _, opt := range req.Options {
			if err := tx.Create(&topicModel.TopicPollOption{
				Text:   opt.Text,
				PollID: poll.ID,
			}).Error; err != nil {
				return err
			}
		}

		return tx.Model(&topicModel.Topic{}).Where("id = ?", req.TopicID).
			Updates(map[string]any{"status_update_time": time.Now()}).Error
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

	uid := 0
	role := 0
	if userInfo != nil {
		uid = userInfo.UID
		role = userInfo.Role
	}

	var responses []dto.TopicPollResponse
	for _, poll := range polls {
		resp := s.buildPollResponse(&poll, uid, role)
		responses = append(responses, resp)
	}

	if responses == nil {
		responses = []dto.TopicPollResponse{}
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
			return errors.ErrBadRequest("至少选择 " + itoa(poll.MinChoice) + " 个选项")
		}
		if len(req.OptionIDArray) > poll.MaxChoice {
			return errors.ErrBadRequest("最多选择 " + itoa(poll.MaxChoice) + " 个选项")
		}
	}

	// Check previous votes
	hasVoted, _ := s.pollRepo.HasUserVoted(req.PollID, uid)
	if hasVoted && !poll.CanChangeVote {
		return errors.ErrBadRequest("该投票不允许修改投票结果")
	}

	txErr := s.pollRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Delete previous votes and decrement counts
		if hasVoted {
			oldOptionIDs, _ := s.pollRepo.FindUserVoteOptionIDs(req.PollID, uid)
			tx.Where("poll_id = ? AND user_id = ?", req.PollID, uid).
				Delete(&topicModel.TopicPollVote{})
			for _, oid := range oldOptionIDs {
				tx.Model(&topicModel.TopicPollOption{}).Where("id = ?", oid).
					Update("vote_count", gorm.Expr("vote_count - 1"))
			}
		}

		// Create new votes and increment counts
		for _, optionID := range req.OptionIDArray {
			tx.Create(&topicModel.TopicPollVote{
				PollID:   req.PollID,
				OptionID: optionID,
				UserID:   uid,
			})
			tx.Model(&topicModel.TopicPollOption{}).Where("id = ?", optionID).
				Update("vote_count", gorm.Expr("vote_count + 1"))
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
		tx.Where("poll_id = ?", pollID).Delete(&topicModel.TopicPollVote{})
		tx.Where("poll_id = ?", pollID).Delete(&topicModel.TopicPollOption{})
		return tx.Delete(&topicModel.TopicPoll{}, pollID).Error
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

	uid := 0
	role := 0
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

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

func (s *PollService) buildPollResponse(poll *topicModel.TopicPoll, uid, role int) dto.TopicPollResponse {
	options, _ := s.pollRepo.FindOptionsByPollID(poll.ID)
	hasVoted, _ := s.pollRepo.HasUserVoted(poll.ID, uid)
	canView := canViewResults(poll, uid, role, hasVoted)

	var userVotedOptionIDs map[int]bool
	if uid > 0 {
		votedIDs, _ := s.pollRepo.FindUserVoteOptionIDs(poll.ID, uid)
		userVotedOptionIDs = make(map[int]bool, len(votedIDs))
		for _, id := range votedIDs {
			userVotedOptionIDs[id] = true
		}
	}

	optionResponses := make([]dto.PollOptionResponse, len(options))
	for i, opt := range options {
		var voteCount *int
		if canView {
			vc := opt.VoteCount
			voteCount = &vc
		}
		optionResponses[i] = dto.PollOptionResponse{
			ID:        opt.ID,
			Text:      opt.Text,
			VoteCount: voteCount,
			IsVoted:   userVotedOptionIDs[opt.ID],
		}
	}

	var voters []dto.KunUser
	var votersCount int
	var totalVoteCount *int
	if canView {
		if !poll.IsAnonymous {
			voters, _ = s.pollRepo.FindDistinctVoters(poll.ID, 5)
		}
		vc, _ := s.pollRepo.CountDistinctVoters(poll.ID)
		votersCount = vc
		tc, _ := s.pollRepo.CountTotalVotes(poll.ID)
		totalVoteCount = &tc
	}
	if voters == nil {
		voters = []dto.KunUser{}
	}

	// Fetch poll creator user info
	var creator dto.KunUser
	s.pollRepo.DB().Table(`"user"`).Select("id, name, avatar").
		Where("id = ?", poll.UserID).Scan(&creator)

	return dto.TopicPollResponse{
		ID: poll.ID, Title: poll.Title, Description: poll.Description,
		MinChoice: poll.MinChoice, MaxChoice: poll.MaxChoice,
		Deadline: poll.Deadline, Type: poll.Type, Status: poll.Status,
		ResultVisibility: poll.ResultVisibility,
		IsAnonymous: poll.IsAnonymous, CanChangeVote: poll.CanChangeVote,
		TopicID: poll.TopicID, Created: poll.CreatedAt, Updated: poll.UpdatedAt,
		User: creator, Options: optionResponses,
		HasVoted: hasVoted, Voters: voters,
		VotersCount: votersCount, VoteCount: totalVoteCount,
	}
}

func canViewResults(poll *topicModel.TopicPoll, uid, role int, hasVoted bool) bool {
	if uid == poll.UserID || role > 1 {
		return true
	}
	isPollFinished := poll.Status == "closed" ||
		(poll.Deadline != nil && time.Now().After(*poll.Deadline))

	switch poll.ResultVisibility {
	case "always":
		return true
	case "after_vote":
		return hasVoted
	case "after_deadline":
		return isPollFinished
	default:
		return false
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
