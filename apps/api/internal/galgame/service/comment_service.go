package service

import (
	"context"
	"fmt"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	msgModel "kun-galgame-api/internal/message/model"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	stateRepo   *userRepo.StateRepository
	userClient  *userclient.Client
	helpers     InteractionHelpers
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	stateRepo *userRepo.StateRepository,
	userClient *userclient.Client,
) *CommentService {
	return &CommentService{commentRepo: commentRepo, stateRepo: stateRepo, userClient: userClient}
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

func (s *CommentService) GetComments(ctx context.Context, galgameID, page, limit int) *CommentListResult {
	total := s.commentRepo.CountByGalgame(galgameID)
	rows := s.commentRepo.FindPaginated(galgameID, page, limit)

	// Collect every uid we need to render (authors + targets).
	uidSet := make(map[int]struct{})
	for _, r := range rows {
		uidSet[r.UserID] = struct{}{}
		if r.TargetUserID != nil && *r.TargetUserID > 0 {
			uidSet[*r.TargetUserID] = struct{}{}
		}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		uids = append(uids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]CommentItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		item := CommentItem{
			ID: r.ID, Content: r.Content, GalgameID: r.GalgameID,
			User:      UserObj{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			LikeCount: r.LikeCount, Created: r.CreatedAt,
		}
		if r.TargetUserID != nil {
			t := userMap[*r.TargetUserID]
			item.TargetUser = &UserObj{
				ID: t.ID, Name: t.Name, Avatar: t.Avatar,
			}
		}
		items = append(items, item)
	}

	return &CommentListResult{Items: items, Total: total}
}

// ──────────────────────────────────────────
// CreateComment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	ctx context.Context,
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
			if err := s.stateRepo.AdjustMoemoepointTx(tx, *targetUserID, 1); err != nil {
				return err
			}

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

	// Build response — identity from OAuth.
	creator, _, _ := s.userClient.User(ctx, uid)
	resp := &CommentItem{
		ID: comment.ID, Content: comment.Content, GalgameID: comment.GalgameID,
		User:      UserObj{ID: creator.ID, Name: creator.Name, Avatar: creator.Avatar},
		LikeCount: 0, Created: comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if targetUserID != nil {
		t, _, _ := s.userClient.User(ctx, *targetUserID)
		resp.TargetUser = &UserObj{ID: t.ID, Name: t.Name, Avatar: t.Avatar}
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
				if err := s.stateRepo.AdjustMoemoepointTx(tx, comment.UserID, 1); err != nil {
					return err
				}
				s.helpers.CreateGalgameMessageWithContent(
					tx, uid, comment.UserID, "liked",
					truncate(comment.Content, 233),
					comment.GalgameID,
				)
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameComment{}).Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count - 1"))
			if comment.UserID != uid {
				if err := s.stateRepo.AdjustMoemoepointTx(tx, comment.UserID, -1); err != nil {
					return err
				}
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
