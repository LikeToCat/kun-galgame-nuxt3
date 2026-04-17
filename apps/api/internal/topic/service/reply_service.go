package service

import (
	"context"
	"fmt"
	"strings"
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

type ReplyService struct {
	replyRepo   *repository.ReplyRepository
	commentRepo *repository.CommentRepository
	topicRepo   *repository.TopicRepository
	rdb         *redis.Client
	helpers     InteractionHelpers
}

func NewReplyService(
	replyRepo *repository.ReplyRepository,
	commentRepo *repository.CommentRepository,
	topicRepo *repository.TopicRepository,
	rdb *redis.Client,
) *ReplyService {
	return &ReplyService{
		replyRepo:   replyRepo,
		commentRepo: commentRepo,
		topicRepo:   topicRepo,
		rdb:         rdb,
	}
}

// ──────────────────────────────────────────
// List replies
// ──────────────────────────────────────────

func (s *ReplyService) GetReplies(
	ctx context.Context,
	req *dto.ListRepliesRequest,
	userInfo *middleware.UserInfo,
) ([]dto.TopicReplyResponse, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(req.TopicID)
	if err != nil {
		return []dto.TopicReplyResponse{}, nil
	}

	// Collect special reply IDs (pinned + best answer)
	var specialIDs []int
	if topic.PinnedReplyID != nil {
		specialIDs = append(specialIDs, *topic.PinnedReplyID)
	}
	if topic.BestAnswerID != nil && (topic.PinnedReplyID == nil || *topic.BestAnswerID != *topic.PinnedReplyID) {
		specialIDs = append(specialIDs, *topic.BestAnswerID)
	}

	var result []dto.TopicReplyResponse

	// On page 1, prepend special replies
	if req.Page == 1 && len(specialIDs) > 0 {
		if specialRows, err := s.replyRepo.FindRepliesByIDs(specialIDs); err == nil {
			result = append(result, s.buildReplyResponses(specialRows, topic, userInfo)...)
		}
	}

	regularRows, err := s.replyRepo.FindRepliesPaginated(
		req.TopicID, specialIDs,
		req.Page, req.Limit, req.SortOrder,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取回复列表失败")
	}

	result = append(result, s.buildReplyResponses(regularRows, topic, userInfo)...)

	if result == nil {
		result = []dto.TopicReplyResponse{}
	}
	return result, nil
}

// ──────────────────────────────────────────
// Reply detail
// ──────────────────────────────────────────

func (s *ReplyService) GetReplyDetail(
	ctx context.Context,
	replyID int,
	userInfo *middleware.UserInfo,
) (*dto.TopicReplyResponse, *errors.AppError) {
	rows, err := s.replyRepo.FindRepliesByIDs([]int{replyID})
	if err != nil || len(rows) == 0 {
		return nil, errors.ErrNotFound("未找到该回复")
	}

	topic, _ := s.topicRepo.FindByID(rows[0].TopicID)
	responses := s.buildReplyResponses(rows, topic, userInfo)
	if len(responses) == 0 {
		return nil, errors.ErrNotFound("未找到该回复")
	}
	return &responses[0], nil
}

// ──────────────────────────────────────────
// Create reply — floor calculation inside tx
// ──────────────────────────────────────────

func (s *ReplyService) CreateReply(
	ctx context.Context,
	uid int,
	req *dto.CreateReplyRequest,
) (*dto.TopicReplyResponse, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(req.TopicID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该话题")
	}

	validTargets := make([]dto.ReplyTarget, 0, len(req.Targets))
	for _, t := range req.Targets {
		if strings.TrimSpace(t.Content) != "" {
			validTargets = append(validTargets, t)
		}
	}

	if strings.TrimSpace(req.Content) == "" && len(validTargets) == 0 {
		return nil, errors.ErrBadRequest("回复内容不能为空")
	}

	var newReply *topicModel.TopicReply

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		maxFloor, err := s.replyRepo.GetMaxFloor(tx, req.TopicID)
		if err != nil {
			return err
		}

		newReply = &topicModel.TopicReply{
			UserID:  uid,
			TopicID: req.TopicID,
			Floor:   maxFloor + 1,
			Content: req.Content,
		}
		if err := s.replyRepo.CreateReply(tx, newReply); err != nil {
			return err
		}

		for _, t := range validTargets {
			if err := s.replyRepo.CreateReplyTarget(tx, &topicModel.TopicReplyTarget{
				ReplyID:       newReply.ID,
				TargetReplyID: t.TargetReplyID,
				Content:       t.Content,
			}); err != nil {
				return err
			}
		}

		if err := s.topicRepo.TouchStatusUpdateTime(tx, req.TopicID, time.Now()); err != nil {
			return err
		}

		// Collect distinct target users (minus self)
		targetUserSet := make(map[int]bool)
		for _, t := range validTargets {
			targetUID, err := s.replyRepo.FindTargetReplyUserID(tx, t.TargetReplyID)
			if err == nil && targetUID != uid {
				targetUserSet[targetUID] = true
			}
		}

		preview := truncate(req.Content, constants.TextPreviewLength)

		for targetUID := range targetUserSet {
			s.helpers.AdjustMoemoepoint(tx, targetUID, constants.RewardReply)
			s.helpers.CreateReplyMessage(tx, uid, targetUID, "replied", preview, req.TopicID)
		}

		// Reward topic owner (matches original: always creates an extra
		// "replied" message even if owner is already a target recipient).
		if strings.TrimSpace(req.Content) != "" && topic.UserID != uid {
			s.helpers.AdjustMoemoepoint(tx, topic.UserID, constants.RewardReply)
			s.helpers.CreateReplyMessage(tx, uid, topic.UserID, "replied", preview, req.TopicID)
		}

		return nil
	})

	if txErr != nil {
		return nil, errors.ErrInternal("创建回复失败")
	}

	rows, _ := s.replyRepo.FindRepliesByIDs([]int{newReply.ID})
	if len(rows) == 0 {
		return nil, errors.ErrInternal("创建回复失败")
	}
	responses := s.buildReplyResponses(rows, topic, nil)
	return &responses[0], nil
}

// ──────────────────────────────────────────
// Update reply
// ──────────────────────────────────────────

func (s *ReplyService) UpdateReply(
	ctx context.Context,
	uid int,
	req *dto.UpdateReplyRequest,
) *errors.AppError {
	reply, err := s.replyRepo.FindByID(req.ReplyID)
	if err != nil {
		return errors.ErrNotFound("未找到该回复")
	}
	if reply.UserID != uid {
		return errors.ErrForbidden("您没有权限编辑此回复")
	}

	now := time.Now()
	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.replyRepo.UpdateReplyContent(tx, req.ReplyID, map[string]any{
			"content": req.Content,
			"edited":  &now,
		}); err != nil {
			return err
		}

		if len(req.Targets) > 0 {
			if err := s.replyRepo.DeleteReplyTargetsByReplyID(tx, req.ReplyID); err != nil {
				return err
			}
			for _, t := range req.Targets {
				if strings.TrimSpace(t.Content) == "" {
					continue
				}
				if err := s.replyRepo.CreateReplyTarget(tx, &topicModel.TopicReplyTarget{
					ReplyID:       req.ReplyID,
					TargetReplyID: t.TargetReplyID,
					Content:       t.Content,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if txErr != nil {
		return errors.ErrInternal("更新回复失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete reply — cascade + moemoepoint penalty
// ──────────────────────────────────────────

func (s *ReplyService) DeleteReply(
	ctx context.Context,
	uid, role, replyID int,
) *errors.AppError {
	reply, err := s.replyRepo.FindByID(replyID)
	if err != nil {
		return errors.ErrNotFound("未找到该回复")
	}
	if reply.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此回复")
	}

	commentCount, likeCount, targetCount, targetByCount, _ := s.replyRepo.CountReplyRelated(replyID)

	penalty := 3
	if reply.UserID == uid && role < 2 {
		penalty = 3 * int(commentCount+likeCount+targetCount+targetByCount+1)
	}

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		user, err := s.replyRepo.LockUserForUpdate(tx, reply.UserID)
		if err != nil {
			return err
		}
		if user.Moemoepoint < penalty {
			return gorm.ErrCheckConstraintViolated
		}

		allIDs, err := s.replyRepo.CollectCascadeReplyIDs(tx, []int{replyID})
		if err != nil {
			return err
		}
		if err := s.replyRepo.DeleteRepliesByIDs(tx, allIDs); err != nil {
			return err
		}

		s.helpers.AdjustMoemoepoint(tx, reply.UserID, -penalty)
		return nil
	})

	if txErr == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 无法删除此回复")
	}
	if txErr != nil {
		return errors.ErrInternal("删除回复失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Reply interactions
// ──────────────────────────────────────────

func (s *ReplyService) ToggleReplyLike(ctx context.Context, uid, replyID int) *errors.AppError {
	err := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		reply, err := s.replyRepo.FindByIDTx(tx, replyID)
		if err != nil {
			return err
		}
		if reply.UserID == uid {
			return gorm.ErrInvalidData
		}

		existing, findErr := s.replyRepo.FindReplyLike(tx, uid, replyID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.replyRepo.CreateReplyLike(tx, uid, replyID); err != nil {
				return err
			}
			if err := s.replyRepo.AdjustReplyLikeCount(tx, replyID, 1); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, reply.UserID, 1)

			link := fmt.Sprintf("/topic/%d", reply.TopicID)
			preview := truncate(reply.Content, constants.TextPreviewLength)
			createDedupMessage(tx, uid, reply.UserID, "liked", preview, link)
		} else if findErr == nil {
			if err := s.replyRepo.DeleteReplyLike(tx, existing); err != nil {
				return err
			}
			if err := s.replyRepo.AdjustReplyLikeCount(tx, replyID, -1); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, reply.UserID, -1)
		} else {
			return findErr
		}
		return nil
	})

	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能给自己的回复点赞")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *ReplyService) ToggleReplyDislike(ctx context.Context, uid, replyID int) *errors.AppError {
	err := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		reply, err := s.replyRepo.FindByIDTx(tx, replyID)
		if err != nil {
			return err
		}
		if reply.UserID == uid {
			return gorm.ErrInvalidData
		}

		existing, findErr := s.replyRepo.FindReplyDislike(tx, uid, replyID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.replyRepo.CreateReplyDislike(tx, uid, replyID); err != nil {
				return err
			}
			return s.replyRepo.AdjustReplyDislikeCount(tx, replyID, 1)
		} else if findErr == nil {
			if err := s.replyRepo.DeleteReplyDislike(tx, existing); err != nil {
				return err
			}
			return s.replyRepo.AdjustReplyDislikeCount(tx, replyID, -1)
		}
		return findErr
	})

	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能踩自己的回复")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *ReplyService) PinReply(ctx context.Context, uid, role, topicID, replyID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限置顶回复")
	}

	var newPinned *int
	if topic.PinnedReplyID != nil && *topic.PinnedReplyID == replyID {
		newPinned = nil // unpin
	} else {
		newPinned = &replyID
	}

	if err := s.topicRepo.UpdateFields(topicID, map[string]any{"pinned_reply_id": newPinned}); err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}
