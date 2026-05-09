package service

import (
	"context"
	"fmt"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CommentService struct {
	replyRepo   *repository.ReplyRepository
	commentRepo *repository.CommentRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	rdb         *redis.Client
	helpers     InteractionHelpers
}

func NewCommentService(
	replyRepo *repository.ReplyRepository,
	commentRepo *repository.CommentRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
	rdb *redis.Client,
) *CommentService {
	return &CommentService{
		replyRepo: replyRepo, commentRepo: commentRepo,
		stateRepo: stateRepo, userClient: userClient, rdb: rdb,
	}
}

// ──────────────────────────────────────────
// Create comment
// ──────────────────────────────────────────

// CreateComment inserts a new comment and returns it in the frontend-expected
// shape (full user + targetUser objects). Returning just a message body made
// the frontend push `undefined` into its comment list, crashing on
// `comment.user.name`.
func (s *CommentService) CreateComment(
	ctx context.Context,
	uid int,
	topicID, replyID, targetUserID int,
	content string,
) (*dto.TopicCommentResponse, *errors.AppError) {
	comment := &topicModel.TopicComment{
		TopicID:      topicID,
		TopicReplyID: replyID,
		UserID:       uid,
		TargetUserID: targetUserID,
		Content:      content,
	}

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
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
		return nil, errors.ErrInternal("发表评论失败")
	}

	// Resolve author + target in one batch via OAuth so the response carries
	// the fields the frontend TopicComment type declares.
	userMap := s.userClient.Hydrate(ctx, []int{uid, targetUserID})
	author := userMap[uid]
	target := userMap[targetUserID]

	return &dto.TopicCommentResponse{
		ID:         comment.ID,
		ReplyID:    comment.TopicReplyID,
		TopicID:    comment.TopicID,
		User:       dto.KunUser{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
		TargetUser: dto.KunUser{ID: target.ID, Name: target.Name, Avatar: target.Avatar},
		Content:    comment.Content,
		IsLiked:    false,
		LikeCount:  0,
		Created:    comment.CreatedAt,
	}, nil
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
		state, err := s.stateRepo.LockForUpdate(tx, comment.UserID)
		if err != nil {
			return err
		}
		if state.Moemoepoint < penalty {
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
