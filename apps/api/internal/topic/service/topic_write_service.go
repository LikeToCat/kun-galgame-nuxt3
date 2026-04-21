package service

import (
	"context"
	"time"

	"kun-galgame-api/internal/constants"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type TopicWriteService struct {
	topicRepo    *repository.TopicRepository
	taxonomyRepo *repository.TopicTaxonomyRepository
	replyRepo    *repository.ReplyRepository
	rdb          *redis.Client
	notifier     msgService.Notifier
	helpers      InteractionHelpers
}

func NewTopicWriteService(
	topicRepo *repository.TopicRepository,
	taxonomyRepo *repository.TopicTaxonomyRepository,
	replyRepo *repository.ReplyRepository,
	rdb *redis.Client,
	notifier msgService.Notifier,
) *TopicWriteService {
	return &TopicWriteService{
		topicRepo:    topicRepo,
		taxonomyRepo: taxonomyRepo,
		replyRepo:    replyRepo,
		rdb:          rdb,
		notifier:     notifier,
	}
}

// ──────────────────────────────────────────
// Create — all checks inside transaction
// ──────────────────────────────────────────

func (s *TopicWriteService) Create(
	ctx context.Context,
	uid int,
	req *dto.CreateTopicRequest,
) (int, *errors.AppError) {
	hasConsumeSection := false
	for _, sec := range req.Sections {
		if constants.TopicSectionConsume[sec] {
			hasConsumeSection = true
			break
		}
	}

	var newTopicID int

	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		user, err := s.topicRepo.LockUserForUpdate(tx, uid)
		if err != nil {
			return err
		}

		todayCount, err := s.topicRepo.CountTodayTopicsByUser(tx, uid)
		if err != nil {
			return err
		}
		dailyLimit := int64(user.Moemoepoint/10 + 1)
		if todayCount >= dailyLimit {
			return gorm.ErrInvalidData
		}

		if hasConsumeSection && user.Moemoepoint < constants.CostConsumeSection {
			return gorm.ErrInvalidData
		}

		topic := &topicModel.Topic{
			Title:    req.Title,
			Content:  req.Content,
			Category: req.Category,
			IsNSFW:   req.IsNSFW,
			UserID:   uid,
		}
		if err := s.topicRepo.CreateTopic(tx, topic); err != nil {
			return err
		}
		newTopicID = topic.ID

		tags, err := s.taxonomyRepo.FindOrCreateTags(req.Tags)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			if err := s.taxonomyRepo.CreateTopicTagRelation(tx, topic.ID, tag.ID); err != nil {
				return err
			}
		}

		sections, err := s.taxonomyRepo.FindSectionsByNamesTx(tx, req.Sections)
		if err != nil {
			return err
		}
		for _, sec := range sections {
			if err := s.taxonomyRepo.CreateSectionRelation(tx, topic.ID, sec.ID); err != nil {
				return err
			}
		}

		pointsDelta := constants.RewardCreateTopic
		if hasConsumeSection {
			pointsDelta = -constants.CostConsumeSection
		}
		s.helpers.AdjustMoemoepoint(tx, uid, pointsDelta)
		return nil
	})

	if err != nil {
		if err == gorm.ErrInvalidData {
			if hasConsumeSection {
				return 0, errors.ErrBadRequest("您的萌萌点不足, 无法发布此类型话题")
			}
			return 0, errors.ErrBadRequest("您今日发布的话题已达上限")
		}
		return 0, errors.ErrInternal("创建话题失败")
	}

	return newTopicID, nil
}

// ──────────────────────────────────────────
// Update
// ──────────────────────────────────────────

func (s *TopicWriteService) Update(
	ctx context.Context,
	uid, role int,
	topicID int,
	req *dto.UpdateTopicRequest,
) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限编辑此话题")
	}

	now := time.Now()
	txErr := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.topicRepo.UpdateTopicFields(tx, topicID, map[string]any{
			"title":              req.Title,
			"content":            req.Content,
			"category":           req.Category,
			"is_nsfw":            req.IsNSFW,
			"edited":             &now,
			"status_update_time": now,
		}); err != nil {
			return err
		}

		tags, err := s.taxonomyRepo.FindOrCreateTags(req.Tags)
		if err != nil {
			return err
		}
		tagIDs := make([]int, len(tags))
		for i, t := range tags {
			tagIDs[i] = t.ID
		}
		if err := s.taxonomyRepo.ReplaceTopicTags(tx, topicID, tagIDs); err != nil {
			return err
		}

		sections, err := s.taxonomyRepo.FindSectionsByNamesTx(tx, req.Sections)
		if err != nil {
			return err
		}
		sectionIDs := make([]int, len(sections))
		for i, sec := range sections {
			sectionIDs[i] = sec.ID
		}
		return s.taxonomyRepo.ReplaceSectionRelations(tx, topicID, sectionIDs)
	})

	if txErr != nil {
		return errors.ErrInternal("更新话题失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Interactions — all checks inside transaction
// ──────────────────────────────────────────

func (s *TopicWriteService) ToggleLike(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		topic, err := s.topicRepo.FindByIDTx(tx, topicID)
		if err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}
		if topic.UserID == uid {
			return gorm.ErrInvalidData
		}

		existing, findErr := s.topicRepo.FindTopicLike(tx, uid, topicID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.topicRepo.CreateTopicLike(tx, uid, topicID); err != nil {
				return err
			}
			if err := s.topicRepo.AdjustLikeCount(tx, topicID, 1); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, topic.UserID, 1)
			s.helpers.CreateTopicMessage(tx, uid, topic.UserID, "liked", topicID)
		} else if findErr == nil {
			if err := s.topicRepo.DeleteTopicLike(tx, existing); err != nil {
				return err
			}
			if err := s.topicRepo.AdjustLikeCount(tx, topicID, -1); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, topic.UserID, -1)
		} else {
			return findErr
		}
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能给自己点赞")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicWriteService) ToggleDislike(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		topic, err := s.topicRepo.FindByIDTx(tx, topicID)
		if err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}

		existing, findErr := s.topicRepo.FindTopicDislike(tx, uid, topicID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.topicRepo.CreateTopicDislike(tx, uid, topicID); err != nil {
				return err
			}
			return s.topicRepo.AdjustDislikeCount(tx, topicID, 1)
		} else if findErr == nil {
			if err := s.topicRepo.DeleteTopicDislike(tx, existing); err != nil {
				return err
			}
			return s.topicRepo.AdjustDislikeCount(tx, topicID, -1)
		}
		return findErr
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicWriteService) Upvote(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		topic, err := s.topicRepo.FindByIDTx(tx, topicID)
		if err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}
		if topic.UserID == uid {
			return gorm.ErrInvalidData
		}

		user, err := s.topicRepo.LockUserForUpdate(tx, uid)
		if err != nil {
			return err
		}
		if user.Moemoepoint < constants.CostUpvoteSender {
			return gorm.ErrCheckConstraintViolated
		}

		now := time.Now()

		if err := s.topicRepo.CreateTopicUpvote(tx, uid, topicID); err != nil {
			return err
		}
		if err := s.topicRepo.ApplyUpvoteCountAndTime(tx, topicID, now); err != nil {
			return err
		}

		s.helpers.AdjustMoemoepoint(tx, uid, -constants.CostUpvoteSender)
		s.helpers.AdjustMoemoepoint(tx, topic.UserID, constants.RewardUpvoteOwner)
		s.helpers.CreateTopicMessage(tx, uid, topic.UserID, "upvoted", topicID)
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能推自己的话题")
	}
	if err == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 推话题需要 7 萌萌点")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicWriteService) ToggleFavorite(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		topic, err := s.topicRepo.FindByIDTx(tx, topicID)
		if err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}

		existing, findErr := s.topicRepo.FindTopicFavorite(tx, uid, topicID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.topicRepo.CreateTopicFavorite(tx, uid, topicID); err != nil {
				return err
			}
			if err := s.topicRepo.AdjustFavoriteCount(tx, topicID, 1); err != nil {
				return err
			}
			if uid != topic.UserID {
				s.helpers.AdjustMoemoepoint(tx, topic.UserID, 1)
				s.helpers.CreateTopicMessage(tx, uid, topic.UserID, "favorite", topicID)
			}
		} else if findErr == nil {
			if err := s.topicRepo.DeleteTopicFavorite(tx, existing); err != nil {
				return err
			}
			if err := s.topicRepo.AdjustFavoriteCount(tx, topicID, -1); err != nil {
				return err
			}
			if uid != topic.UserID {
				s.helpers.AdjustMoemoepoint(tx, topic.UserID, -1)
			}
		} else {
			return findErr
		}
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicWriteService) ToggleHide(ctx context.Context, uid, role, topicID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限操作此话题")
	}

	newStatus := 1
	if topic.Status == 1 {
		newStatus = 0
	}
	if err := s.topicRepo.UpdateFields(topicID, map[string]any{"status": newStatus}); err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// SetBestAnswer toggles the best answer for a topic: if the given reply is
// already the current best answer it is cleared, otherwise it becomes the
// best answer. The reply author's moemoepoint is adjusted by ±7 to match
// the legacy Nitro behavior.
func (s *TopicWriteService) SetBestAnswer(ctx context.Context, uid, role, topicID, replyID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("只有话题作者或管理员可以设置最佳回答")
	}

	var reply topicModel.TopicReply
	if err := s.topicRepo.DB().First(&reply, replyID).Error; err != nil {
		return errors.ErrNotFound("未找到该回复")
	}
	if reply.TopicID != topicID {
		return errors.ErrBadRequest("该回复不属于此话题")
	}

	isCurrentBest := topic.BestAnswerID != nil && *topic.BestAnswerID == replyID
	delta := 7
	if isCurrentBest {
		delta = -7
	}

	txErr := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		if isCurrentBest {
			if err := tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("best_answer_id", nil).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Updates(map[string]any{
					"best_answer_id":     &replyID,
					"status_update_time": time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec(
			`UPDATE "user" SET moemoepoint = moemoepoint + ? WHERE id = ?`,
			delta, reply.UserID,
		).Error; err != nil {
			return err
		}
		// Only notify on set (not on clear) — matches legacy Nitro.
		if !isCurrentBest {
			return s.notifier.Emit(tx, msgService.Spec{
				SenderID:   uid,
				ReceiverID: reply.UserID,
				Kind:       msgService.NotifySolution,
				Content:    replyPlainPreview(s.replyRepo, reply),
				TopicID:    topicID,
			})
		}
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("设置最佳回答失败")
	}
	return nil
}
