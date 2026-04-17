package service

import (
	"fmt"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	toolsetRepo *repository.ToolsetRepository
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	toolsetRepo *repository.ToolsetRepository,
) *CommentService {
	return &CommentService{commentRepo: commentRepo, toolsetRepo: toolsetRepo}
}

// ──────────────────────────────────────────
// GetComments — GET /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) GetComments(
	toolsetID int,
	req *dto.CommentListRequest,
) ([]dto.CommentItem, int64) {
	total := s.commentRepo.CountByToolset(toolsetID)
	rows := s.commentRepo.FindPaginated(toolsetID, req.Page, req.Limit)

	items := make([]dto.CommentItem, 0, len(rows))
	for _, cm := range rows {
		item := dto.CommentItem{
			GalgameToolsetComment: cm,
			User:                  s.commentRepo.FindUser(cm.UserID),
		}
		// If this is a reply, fetch parent comment's user.
		if cm.ParentID != nil {
			if parent, err := s.commentRepo.FindByID(*cm.ParentID); err == nil {
				pu := s.commentRepo.FindUser(parent.UserID)
				item.ParentUser = &pu
			}
		}
		items = append(items, item)
	}

	return items, total
}

// GetLatestForDetail returns the latest N comments with user info, shaped for
// the toolset detail response.
func (s *CommentService) GetLatestForDetail(toolsetID, limit int) []dto.CommentDetailItem {
	rows := s.commentRepo.FindLatest(toolsetID, limit)
	items := make([]dto.CommentDetailItem, 0, len(rows))
	for _, cm := range rows {
		items = append(items, dto.CommentDetailItem{
			GalgameToolsetComment: cm,
			User:                  s.commentRepo.FindUser(cm.UserID),
		})
	}
	return items
}

// ──────────────────────────────────────────
// CreateComment — POST /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	userID, toolsetID int,
	req *dto.CreateCommentRequest,
) (*dto.CreatedCommentResponse, *errors.AppError) {
	// Verify toolset exists
	toolset, err := s.toolsetRepo.FindByID(toolsetID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该工具")
	}

	comment := model.GalgameToolsetComment{
		Content:   req.Content,
		UserID:    userID,
		ToolsetID: toolsetID,
		ParentID:  req.ParentID,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	// Send notification to toolset owner or parent comment owner
	go s.notifyCommentReceiver(userID, toolsetID, toolset.UserID, req)

	return &comment, nil
}

// notifyCommentReceiver sends a "commented" or "replied" notification.
// It runs in a goroutine (fire-and-forget) so the HTTP request isn't blocked.
func (s *CommentService) notifyCommentReceiver(
	senderID, toolsetID, toolsetOwnerID int,
	req *dto.CreateCommentRequest,
) {
	receiverID := toolsetOwnerID
	msgType := "commented"

	if req.ParentID != nil {
		if parent, err := s.commentRepo.FindByID(*req.ParentID); err == nil {
			receiverID = parent.UserID
			msgType = "replied"
		}
	}

	if receiverID == senderID || receiverID <= 0 {
		return
	}

	s.commentRepo.DB().Create(&msgModel.Message{
		Content:    markdown.ToPlainText(req.Content, 100),
		Link:       fmt.Sprintf("/toolset/%d", toolsetID),
		Type:       msgType,
		SenderID:   senderID,
		ReceiverID: receiverID,
	})
}

// ──────────────────────────────────────────
// UpdateComment — PUT /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) UpdateComment(
	userID int,
	req *dto.UpdateCommentRequest,
) *errors.AppError {
	comment, err := s.commentRepo.FindByID(req.CommentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID {
		return errors.ErrForbidden("您只能编辑自己的评论")
	}

	now := time.Now()
	s.commentRepo.UpdateContent(comment, req.Content, now)
	return nil
}

// ──────────────────────────────────────────
// DeleteComment — DELETE /toolset/:id/comment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(
	userID, userRole, toolsetID int,
	req *dto.DeleteCommentRequest,
) *errors.AppError {
	comment, err := s.commentRepo.FindByID(req.CommentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}

	// Load toolset (may or may not exist; we only need its owner for perms).
	var ownerID int
	if t, err := s.toolsetRepo.FindByID(toolsetID); err == nil {
		ownerID = t.UserID
	} else if err != gorm.ErrRecordNotFound {
		// If the lookup fails for some other reason, treat ownerID as 0.
		ownerID = 0
	}

	if comment.UserID != userID && ownerID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	s.commentRepo.Delete(comment)
	return nil
}
