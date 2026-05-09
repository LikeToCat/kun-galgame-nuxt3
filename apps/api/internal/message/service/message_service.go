package service

import (
	"context"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type MessageService struct {
	messageRepo *repository.MessageRepository
	userClient  *userclient.Client
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	userClient *userclient.Client,
) *MessageService {
	return &MessageService{messageRepo: messageRepo, userClient: userClient}
}

func (s *MessageService) GetMessages(
	ctx context.Context,
	uid int,
	req *dto.ListMessagesRequest,
) (*dto.MessageListResponse, *errors.AppError) {
	rows, total, err := s.messageRepo.FindMessages(
		uid, req.Type, req.SortOrder, req.Page, req.Limit,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取消息列表失败")
	}

	uids := userclient.CollectIDs(rows, func(r repository.MessageRow) int { return r.SenderID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.MessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.SenderID]
		// Hide messages whose sender is banned/deleted.
		if !userclient.IsRenderable(u) {
			continue
		}
		messages = append(messages, dto.MessageResponse{
			ID:          r.ID,
			Sender:      dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ReceiverUID: r.ReceiverID,
			Link:        r.Link,
			Content:     r.Content,
			Status:      r.Status,
			Type:        r.Type,
		})
	}

	return &dto.MessageListResponse{Messages: messages, TotalCount: total}, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, uid, messageID int) *errors.AppError {
	if err := s.messageRepo.DeleteByIDAndReceiver(messageID, uid); err != nil {
		return errors.ErrInternal("删除消息失败")
	}
	return nil
}

func (s *MessageService) MarkAllRead(ctx context.Context, uid int) *errors.AppError {
	if err := s.messageRepo.MarkAllRead(uid); err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	return nil
}

func (s *MessageService) GetSystemMessages(ctx context.Context) ([]dto.SystemMessageResponse, *errors.AppError) {
	rows, err := s.messageRepo.FindSystemMessages()
	if err != nil {
		return nil, errors.ErrInternal("获取系统消息失败")
	}

	uids := userclient.CollectIDs(rows, func(r repository.SystemMessageRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	messages := make([]dto.SystemMessageResponse, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		messages = append(messages, dto.SystemMessageResponse{
			ID:     r.ID,
			Status: r.Status,
			Content: map[string]string{
				"en-us": r.ContentEnUS,
				"ja-jp": r.ContentJaJP,
				"zh-cn": r.ContentZhCN,
				"zh-tw": r.ContentZhTW,
			},
			Admin: dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
		})
	}
	return messages, nil
}

func (s *MessageService) MarkAllSystemRead(ctx context.Context) *errors.AppError {
	if err := s.messageRepo.MarkAllSystemRead(); err != nil {
		return errors.ErrInternal("标记已读失败")
	}
	return nil
}

func (s *MessageService) GetNavSummary(ctx context.Context, uid int) ([]map[string]any, *errors.AppError) {
	result, err := s.messageRepo.GetNavSummary(uid)
	if err != nil {
		return nil, errors.ErrInternal("获取消息概要失败")
	}
	return result, nil
}
