package service

import (
	"context"
	"fmt"

	"kun-galgame-api/internal/constants"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CommentService struct {
	replyRepo   *repository.ReplyRepository
	commentRepo *repository.CommentRepository
	rdb         *redis.Client
	helpers     InteractionHelpers
}

func NewCommentService(
	replyRepo *repository.ReplyRepository,
	commentRepo *repository.CommentRepository,
	rdb *redis.Client,
) *CommentService {
	return &CommentService{replyRepo: replyRepo, commentRepo: commentRepo, rdb: rdb}
}

// ──────────────────────────────────────────
// Create comment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	ctx context.Context,
	uid int,
	topicID, replyID, targetUserID int,
	content string,
) *errors.AppError {
	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		comment := &topicModel.TopicComment{
			TopicID:      topicID,
			TopicReplyID: replyID,
			UserID:       uid,
			TargetUserID: targetUserID,
			Content:      content,
		}
		if err := s.commentRepo.CreateComment(tx, comment); err != nil {
			return err
		}

		if uid != targetUserID {
			s.helpers.AdjustMoemoepoint(tx, targetUserID, constants.RewardReply)

			preview := truncate(content, constants.TextPreviewLength)
			s.helpers.CreateReplyMessage(tx, uid, targetUserID, "commented", preview, topicID)
		}
		return nil
	})

	if txErr != nil {
		return errors.ErrInternal("发表评论失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Toggle comment like
// ──────────────────────────────────────────

func (s *CommentService) ToggleCommentLike(ctx context.Context, uid, commentID int) *errors.AppError {
	err := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		comment, err := s.commentRepo.FindCommentByIDTx(tx, commentID)
		if err != nil {
			return err
		}
		if comment.UserID == uid {
			return gorm.ErrInvalidData
		}

		existing, findErr := s.commentRepo.FindCommentLike(tx, uid, commentID)

		if findErr == gorm.ErrRecordNotFound {
			if err := s.commentRepo.CreateCommentLike(tx, uid, commentID); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, comment.UserID, 1)

			link := fmt.Sprintf("/topic/%d", comment.TopicID)
			preview := truncate(comment.Content, constants.TextPreviewLength)
			createDedupMessage(tx, uid, comment.UserID, "liked", preview, link)
		} else if findErr == nil {
			if err := s.commentRepo.DeleteCommentLike(tx, existing); err != nil {
				return err
			}
			s.helpers.AdjustMoemoepoint(tx, comment.UserID, -1)
		} else {
			return findErr
		}
		return nil
	})

	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能给自己的评论点赞")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete comment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(ctx context.Context, uid, role, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindCommentByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	likeCount, _ := s.commentRepo.CountCommentLikes(commentID)
	penalty := 3
	if comment.UserID == uid && role < 2 {
		penalty = 3 * int(likeCount+1)
	}

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		user, err := s.replyRepo.LockUserForUpdate(tx, comment.UserID)
		if err != nil {
			return err
		}
		if user.Moemoepoint < penalty {
			return gorm.ErrCheckConstraintViolated
		}

		if err := s.commentRepo.DeleteCommentLikesForComment(tx, commentID); err != nil {
			return err
		}
		if err := s.commentRepo.DeleteCommentByID(tx, commentID); err != nil {
			return err
		}

		s.helpers.AdjustMoemoepoint(tx, comment.UserID, -penalty)
		return nil
	})

	if txErr == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 无法删除此评论")
	}
	if txErr != nil {
		return errors.ErrInternal("删除评论失败")
	}
	return nil
}
