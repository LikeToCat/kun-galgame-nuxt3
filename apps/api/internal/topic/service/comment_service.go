package service

import (
	"context"
	"fmt"

	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommentService struct {
	replyRepo *repository.ReplyRepository
	rdb       *redis.Client
}

func NewCommentService(
	replyRepo *repository.ReplyRepository,
	rdb *redis.Client,
) *CommentService {
	return &CommentService{replyRepo: replyRepo, rdb: rdb}
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
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		if uid != targetUserID {
			tx.Model(&userModel.User{}).Where("id = ?", targetUserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardReply))

			link := fmt.Sprintf("/topic/%d", topicID)
			tx.Create(&msgModel.Message{
				SenderID:   uid,
				ReceiverID: targetUserID,
				Type:       "commented",
				Content:    truncate(content, constants.TextPreviewLength),
				Link:       link,
				Status:     "unread",
			})
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
		var comment topicModel.TopicComment
		if err := tx.First(&comment, commentID).Error; err != nil {
			return err
		}
		if comment.UserID == uid {
			return gorm.ErrInvalidData
		}

		var existing topicModel.TopicCommentLike
		result := tx.Where("user_id = ? AND topic_comment_id = ?", uid, commentID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&topicModel.TopicCommentLike{UserID: uid, TopicCommentID: commentID})
			tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + 1"))

			link := fmt.Sprintf("/topic/%d", comment.TopicID)
			preview := truncate(comment.Content, constants.TextPreviewLength)
			createDedupMessageByLink(tx, uid, comment.UserID, "liked", preview, link)
		} else {
			tx.Delete(&existing)
			tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
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
	comment, err := s.replyRepo.FindCommentByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	likeCount, _ := s.replyRepo.CountCommentLikes(commentID)
	penalty := 3
	if comment.UserID == uid && role < 2 {
		penalty = 3 * int(likeCount+1)
	}

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		var user userModel.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, comment.UserID).Error; err != nil {
			return err
		}
		if user.Moemoepoint < penalty {
			return gorm.ErrCheckConstraintViolated
		}

		tx.Where("topic_comment_id = ?", commentID).Delete(&topicModel.TopicCommentLike{})
		tx.Delete(&topicModel.TopicComment{}, commentID)

		return tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
			Update("moemoepoint", gorm.Expr("moemoepoint - ?", penalty)).Error
	})

	if txErr == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 无法删除此评论")
	}
	if txErr != nil {
		return errors.ErrInternal("删除评论失败")
	}
	return nil
}
