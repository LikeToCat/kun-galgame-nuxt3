package service

import (
	"context"
	"fmt"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	toolsetRepo *repository.ToolsetRepository
	userClient  *userclient.Client
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	toolsetRepo *repository.ToolsetRepository,
	userClient *userclient.Client,
) *CommentService {
	return &CommentService{commentRepo: commentRepo, toolsetRepo: toolsetRepo, userClient: userClient}
}

// ──────────────────────────────────────────
// GetComments — GET /toolset/:id/comment/all
// Returns the camelCase response shape consumed by the frontend
// (commentData + total, with `targetUser` filled in for replies).
// ──────────────────────────────────────────

func (s *CommentService) GetComments(
	ctx context.Context,
	toolsetID int,
	req *dto.CommentListRequest,
) *dto.ToolsetCommentListResponse {
	total := s.commentRepo.CountByToolset(toolsetID)
	rows := s.commentRepo.FindPaginated(toolsetID, req.Page, req.Limit, req.SortOrder)

	// Batch-resolve unique authors + parent authors to avoid N+1 lookups.
	userIDSet := map[int]struct{}{}
	parentIDs := []int{}
	for _, cm := range rows {
		userIDSet[cm.UserID] = struct{}{}
		if cm.ParentID != nil {
			parentIDs = append(parentIDs, *cm.ParentID)
		}
	}
	parentUserByID := map[int]int{}
	for _, pid := range parentIDs {
		if parent, err := s.commentRepo.FindByID(pid); err == nil {
			parentUserByID[pid] = parent.UserID
			userIDSet[parent.UserID] = struct{}{}
		}
	}
	uids := make([]int, 0, len(userIDSet))
	for id := range userIDSet {
		uids = append(uids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.ToolsetCommentItem, 0, len(rows))
	for _, cm := range rows {
		item := dto.ToolsetCommentItem{
			ID:        cm.ID,
			ToolsetID: cm.ToolsetID,
			Content:   cm.Content,
			Created:   cm.CreatedAt,
			Edited:    cm.Edited,
			ParentID:  cm.ParentID,
			UserID:    cm.UserID,
			Reply:     []dto.ToolsetCommentItem{},
			User:      userBriefFromClient(userMap[cm.UserID]),
		}
		if cm.ParentID != nil {
			if puid, ok := parentUserByID[*cm.ParentID]; ok {
				pu := userBriefFromClient(userMap[puid])
				item.TargetUser = &pu
			}
		}
		items = append(items, item)
	}

	return &dto.ToolsetCommentListResponse{CommentData: items, Total: total}
}

// GetLatestForDetail returns the latest N comments with user info, shaped for
// the toolset detail response.
func (s *CommentService) GetLatestForDetail(ctx context.Context, toolsetID, limit int) []dto.CommentDetailItem {
	rows := s.commentRepo.FindLatest(toolsetID, limit)
	uids := make([]int, 0, len(rows))
	for _, cm := range rows {
		uids = append(uids, cm.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, uids)
	items := make([]dto.CommentDetailItem, 0, len(rows))
	for _, cm := range rows {
		items = append(items, dto.CommentDetailItem{
			GalgameToolsetComment: cm,
			User:                  userBriefFromClient(userMap[cm.UserID]),
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
