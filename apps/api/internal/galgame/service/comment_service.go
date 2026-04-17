package service

import (
	"fmt"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	msgModel "kun-galgame-api/internal/message/model"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
}

func NewCommentService(commentRepo *repository.CommentRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo}
}

// ──────────────────────────────────────────
// Response types
// ──────────────────────────────────────────

type UserObj struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type CommentItem struct {
	ID         int      `json:"id"`
	Content    string   `json:"content"`
	GalgameID  int      `json:"galgameId"`
	User       UserObj  `json:"user"`
	TargetUser *UserObj `json:"targetUser"`
	LikeCount  int      `json:"likeCount"`
	Created    string   `json:"created"`
}

type CommentListResult struct {
	Items []CommentItem
	Total int64
}

// ──────────────────────────────────────────
// GetComments
// ──────────────────────────────────────────

func (s *CommentService) GetComments(galgameID, page, limit int) *CommentListResult {
	total := s.commentRepo.CountByGalgame(galgameID)
	rows := s.commentRepo.FindPaginated(galgameID, page, limit)

	items := make([]CommentItem, len(rows))
	for i, r := range rows {
		item := CommentItem{
			ID: r.ID, Content: r.Content, GalgameID: r.GalgameID,
			User:      UserObj{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			LikeCount: r.LikeCount, Created: r.CreatedAt,
		}
		if r.TargetUserID != nil {
			item.TargetUser = &UserObj{
				ID: *r.TargetUserID, Name: r.TargetUserName, Avatar: r.TargetUserAvatar,
			}
		}
		items[i] = item
	}

	return &CommentListResult{Items: items, Total: total}
}

// ──────────────────────────────────────────
// CreateComment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	uid, galgameID int,
	content string,
	targetUserID *int,
) (*CommentItem, *errors.AppError) {
	comment := model.GalgameComment{
		Content:      content,
		GalgameID:    galgameID,
		UserID:       uid,
		TargetUserID: targetUserID,
	}

	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		tx.Create(&comment)
		tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("comment_count", gorm.Expr("comment_count + 1"))

		if targetUserID != nil && *targetUserID != uid {
			tx.Model(&userModel.User{}).Where("id = ?", *targetUserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + 1"))

			link := fmt.Sprintf("/galgame/%d", galgameID)
			tx.Create(&msgModel.Message{
				SenderID: uid, ReceiverID: *targetUserID,
				Type: "commented", Content: truncate(content, 233),
				Link: link, Status: "unread",
			})
		}
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	// Build response
	creatorName, creatorAvatar := s.commentRepo.GetUserInfo(uid)
	resp := &CommentItem{
		ID: comment.ID, Content: comment.Content, GalgameID: comment.GalgameID,
		User:      UserObj{ID: uid, Name: creatorName, Avatar: creatorAvatar},
		LikeCount: 0, Created: comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if targetUserID != nil {
		targetName, targetAvatar := s.commentRepo.GetUserInfo(*targetUserID)
		resp.TargetUser = &UserObj{ID: *targetUserID, Name: targetName, Avatar: targetAvatar}
	}

	return resp, nil
}

// ──────────────────────────────────────────
// DeleteComment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(uid, role, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		tx.Where("galgame_comment_id = ?", commentID).Delete(&model.GalgameCommentLike{})
		tx.Delete(&comment)
		tx.Model(&model.GalgameLocal{}).Where("id = ?", comment.GalgameID).
			Update("comment_count", gorm.Expr("comment_count - 1"))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除评论失败")
	}

	return nil
}

// ──────────────────────────────────────────
// ToggleCommentLike
// ──────────────────────────────────────────

func (s *CommentService) ToggleCommentLike(uid, commentID int) *errors.AppError {
	txErr := s.commentRepo.DB().Transaction(func(tx *gorm.DB) error {
		var comment model.GalgameComment
		tx.First(&comment, commentID)

		var existing model.GalgameCommentLike
		result := tx.Where("user_id = ? AND galgame_comment_id = ?", uid, commentID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameCommentLike{UserID: uid, CommentID: commentID})
			tx.Model(&model.GalgameComment{}).Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count + 1"))
			if comment.UserID != uid {
				tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameComment{}).Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count - 1"))
			if comment.UserID != uid {
				tx.Model(&userModel.User{}).Where("id = ?", comment.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
			}
		}
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}

	return nil
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
