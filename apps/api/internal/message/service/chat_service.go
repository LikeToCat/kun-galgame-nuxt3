package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/errors"
)

type ChatService struct {
	chatRepo *repository.ChatRepository
}

func NewChatService(chatRepo *repository.ChatRepository) *ChatService {
	return &ChatService{chatRepo: chatRepo}
}

// ──────────────────────────────────────────
// GetNavContact — GET /api/message/nav/contact
// ──────────────────────────────────────────

// GetNavContact returns the chat room list for the message sidebar.
// For private rooms, the display title/avatar/route are resolved to the
// OTHER participant (not the current user).
func (s *ChatService) GetNavContact(ctx context.Context, uid int) ([]dto.NavContactItem, *errors.AppError) {
	rooms, err := s.chatRepo.FindRoomsForUser(uid)
	if err != nil {
		return nil, errors.ErrInternal("查询聊天室失败")
	}
	if len(rooms) == 0 {
		return []dto.NavContactItem{}, nil
	}

	roomIDs := make([]int, len(rooms))
	for i, r := range rooms {
		roomIDs[i] = r.ID
	}

	// Participants: { room_id -> [participant...] }
	roomParts := make(map[int][]repository.ParticipantRow)
	for _, p := range s.chatRepo.FindParticipantsByRoomIDs(roomIDs) {
		roomParts[p.ChatRoomID] = append(roomParts[p.ChatRoomID], p)
	}

	// Unread + total counts per room.
	unreadMap := make(map[int]int)
	for _, u := range s.chatRepo.CountUnreadByRoomIDs(roomIDs, uid) {
		unreadMap[u.ChatRoomID] = u.Count
	}
	totalMap := make(map[int]int)
	for _, t := range s.chatRepo.CountTotalByRoomIDs(roomIDs) {
		totalMap[t.ChatRoomID] = t.Count
	}

	items := make([]dto.NavContactItem, len(rooms))
	for i, r := range rooms {
		title, avatar, route := r.Name, r.Avatar, r.Name
		if r.Type == "private" {
			for _, p := range roomParts[r.ID] {
				if p.UserID != uid {
					title = p.UserName
					avatar = p.UserAvatar
					route = strconv.Itoa(p.UserID)
					break
				}
			}
		}
		items[i] = dto.NavContactItem{
			ChatroomName:    r.Name,
			Content:         r.LastMessageContent,
			LastMessageTime: r.LastMessageTime,
			Count:           totalMap[r.ID],
			UnreadCount:     unreadMap[r.ID],
			Route:           route,
			Title:           title,
			Avatar:          avatar,
		}
	}
	return items, nil
}

// ──────────────────────────────────────────
// GetChatHistory — GET /api/message/chat/history
// ──────────────────────────────────────────

// GetChatHistory returns paginated chat messages between the current user and
// the receiver, in chronological (ascending) order. Side effect: marks every
// fetched message not sent by the current user as read.
func (s *ChatService) GetChatHistory(
	ctx context.Context,
	uid int,
	req *dto.GetChatHistoryRequest,
) ([]dto.ChatMessageItem, *errors.AppError) {
	if req.ReceiverUID == uid {
		return nil, errors.ErrBadRequest("不能给自己发送消息")
	}

	roomID, roomName := s.findOrCreatePrivateRoom(uid, req.ReceiverUID)
	if roomID == 0 {
		return []dto.ChatMessageItem{}, nil
	}

	rows := s.chatRepo.FindMessagesByRoom(roomID, roomName, req.Page, req.Limit)

	// Mark messages received (not sent by current user) as read.
	if len(rows) > 0 {
		msgIDs := make([]int, 0, len(rows))
		for _, m := range rows {
			if m.SenderID != uid {
				msgIDs = append(msgIDs, m.ID)
			}
		}
		s.chatRepo.MarkMessagesRead(msgIDs, uid)
	}

	// Reverse DB order (DESC) into chronological ASC order for the response.
	items := make([]dto.ChatMessageItem, len(rows))
	for i, m := range rows {
		items[len(rows)-1-i] = dto.ChatMessageItem{
			ID:           m.ID,
			ChatroomName: m.ChatroomName,
			Sender:       dto.ChatSender{ID: m.SenderID, Name: m.SenderName, Avatar: m.SenderAvatar},
			ReceiverUID:  m.ReceiverID,
			Content:      m.Content,
			IsRecall:     m.IsRecall,
			Created:      m.Created,
			RecallTime:   m.RecallTime,
			EditTime:     m.EditTime,
			ReadBy:       []dto.ChatSender{},
		}
	}
	return items, nil
}

// ──────────────────────────────────────────
// SendChatMessage — POST /api/message/chat/send
// ──────────────────────────────────────────

// SendChatMessage writes a new chat message and updates the room's
// last_message_* fields. senderName is written into chat_room for display in
// the contacts list; it's passed in because the service doesn't have a user repo.
func (s *ChatService) SendChatMessage(
	ctx context.Context,
	senderUID int,
	senderName string,
	req *dto.SendChatMessageRequest,
) *errors.AppError {
	if req.ReceiverUID == senderUID {
		return errors.ErrBadRequest("不能给自己发送消息")
	}

	roomID, roomName := s.findOrCreatePrivateRoom(senderUID, req.ReceiverUID)
	if roomID == 0 {
		return errors.ErrInternal("创建聊天室失败")
	}

	now := time.Now()
	s.chatRepo.InsertChatMessage(roomID, roomName, senderUID, req.ReceiverUID, req.Content, now)
	s.chatRepo.UpdateRoomLastMessage(roomID, req.Content, senderUID, senderName, now)
	return nil
}

// ──────────────────────────────────────────
// RecallMessage — POST /api/message/chat/recall
// ──────────────────────────────────────────

// RecallMessage marks a chat message as recalled. Only the original sender
// may recall, and only if it hasn't been recalled already. When the recalled
// message was the room's latest, the room's last_message preview is also
// refreshed to "<sender>撤回了一条消息" so contact-list rendering matches the
// chat history view.
func (s *ChatService) RecallMessage(
	ctx context.Context,
	uid int,
	messageID int,
) *errors.AppError {
	header, ok := s.chatRepo.FindMessageHeader(messageID)
	if !ok {
		return errors.ErrNotFound("消息不存在或已被删除")
	}
	if header.SenderID != uid {
		return errors.ErrForbidden("您只能撤回自己发送的消息")
	}
	if header.IsRecall {
		return errors.ErrBadRequest("该消息已被撤回")
	}

	now := time.Now()
	if err := s.chatRepo.MarkMessageRecalled(messageID, now); err != nil {
		return errors.ErrInternal("撤回消息失败")
	}

	// Only refresh the room preview if this WAS the latest message —
	// otherwise the preview should keep showing whatever's actually latest.
	if s.chatRepo.IsLatestMessageInRoom(header.ChatRoomID, messageID) {
		preview := fmt.Sprintf("%s撤回了一条消息", header.SenderName)
		s.chatRepo.UpdateRoomLastMessage(header.ChatRoomID, preview, uid, header.SenderName, now)
	}
	return nil
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

// findOrCreatePrivateRoom returns the (roomID, roomName) for the private room
// between two users, creating it if it doesn't exist. Look-up is by
// participant table (NOT by generated room name, which may be stale after the
// OAuth migration changed user IDs).
func (s *ChatService) findOrCreatePrivateRoom(uid1, uid2 int) (int, string) {
	room := s.chatRepo.FindPrivateRoomBetween(uid1, uid2)
	if room.ID > 0 {
		return room.ID, room.Name
	}
	newRoom := s.chatRepo.CreatePrivateRoom(generateRoomID(uid1, uid2), uid1, uid2)
	return newRoom.ID, newRoom.Name
}

// generateRoomID produces a deterministic "smaller-larger" name for a private
// room between two users.
func generateRoomID(uid1, uid2 int) string {
	if uid1 < uid2 {
		return fmt.Sprintf("%d-%d", uid1, uid2)
	}
	return fmt.Sprintf("%d-%d", uid2, uid1)
}
