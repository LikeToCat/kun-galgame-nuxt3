package service

import (
	"kun-galgame-api/internal/infrastructure/markdown"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/errors"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	websiteRepo *repository.WebsiteRepository
	notifier    msgService.Notifier
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	websiteRepo *repository.WebsiteRepository,
	notifier msgService.Notifier,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		websiteRepo: websiteRepo,
		notifier:    notifier,
	}
}

// ──────────────────────────────────────────
// GetComments — GET /website/:domain/comment
// ──────────────────────────────────────────

// GetComments returns the nested comment tree for a website.
func (s *CommentService) GetComments(websiteID int) []*dto.CommentItem {
	rows := s.commentRepo.FindByWebsite(websiteID)

	flat := make([]*dto.CommentItem, len(rows))
	idMap := make(map[int]*dto.CommentItem, len(rows))
	for i, r := range rows {
		item := &dto.CommentItem{
			ID:        r.ID,
			Content:   r.Content,
			ParentID:  r.ParentID,
			UserID:    r.UserID,
			WebsiteID: websiteID,
			Created:   r.Created,
			Edited:    r.Edited,
			Reply:     []*dto.CommentItem{},
			User: dto.CommentUser{
				ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar,
			},
			TargetUser: nil,
		}
		flat[i] = item
		idMap[r.ID] = item
	}

	var nested []*dto.CommentItem
	for _, item := range flat {
		if item.ParentID != nil {
			if parent, ok := idMap[*item.ParentID]; ok {
				item.TargetUser = parent.User
				parent.Reply = append(parent.Reply, item)
				continue
			}
		}
		nested = append(nested, item)
	}

	if nested == nil {
		nested = []*dto.CommentItem{}
	}
	return nested
}

// ──────────────────────────────────────────
// CreateComment — POST /website/:domain/comment
// ──────────────────────────────────────────

func (s *CommentService) CreateComment(
	userID int,
	req *dto.CreateCommentRequest,
) (*dto.CreatedCommentResponse, *errors.AppError) {
	comment := model.GalgameWebsiteComment{
		Content:   req.Content,
		WebsiteID: req.WebsiteID,
		UserID:    userID,
		ParentID:  req.ParentID,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, errors.ErrInternal("发表评论失败")
	}

	s.websiteRepo.AdjustCommentCount(req.WebsiteID, 1)

	// Notify the parent-comment author (nitro legacy: only when replying
	// to an existing comment, using the website.url slug as the link key).
	if req.ParentID != nil {
		if parent, err := s.commentRepo.FindByID(*req.ParentID); err == nil {
			url := s.websiteRepo.GetURL(req.WebsiteID)
			_ = s.notifier.Emit(nil, msgService.Spec{
				SenderID:   userID,
				ReceiverID: parent.UserID,
				Kind:       msgService.NotifyCommented,
				Content:    markdown.ToPlainText(req.Content, 233),
				WebsiteURL: url,
			})
		}
	}

	return &comment, nil
}

// ──────────────────────────────────────────
// DeleteComment — DELETE /website/:domain/comment
// ──────────────────────────────────────────

func (s *CommentService) DeleteComment(userID, userRole, commentID int) *errors.AppError {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return errors.ErrNotFound("未找到该评论")
	}
	if comment.UserID != userID && userRole < 2 {
		return errors.ErrForbidden("您没有权限删除此评论")
	}

	s.commentRepo.Delete(comment)
	s.websiteRepo.AdjustCommentCount(comment.WebsiteID, -1)
	return nil
}
