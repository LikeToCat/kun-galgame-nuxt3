package handler

import (
	"fmt"
	"strconv"
	"time"

	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MessageHandler struct {
	db             *gorm.DB
	messageService *service.MessageService
}

func NewMessageHandler(db *gorm.DB, messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{db: db, messageService: messageService}
}

// GetMessages returns paginated message list.
// GET /api/message
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.ListMessagesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.messageService.GetMessages(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, result)
}

// DeleteMessage deletes a single message.
// DELETE /api/message/:id
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的消息 ID"))
	}

	if appErr := h.messageService.DeleteMessage(c.Context(), user.UID, id); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "消息已删除")
}

// GetSystemMessages returns all system broadcast messages.
// GET /api/message/admin
func (h *MessageHandler) GetSystemMessages(c *fiber.Ctx) error {
	messages, appErr := h.messageService.GetSystemMessages(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, messages)
}

// MarkAdminRead marks all system messages as read.
// PUT /api/message/admin/read
func (h *MessageHandler) MarkAdminRead(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.messageService.MarkAllSystemRead(c.Context()); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "已标记全部已读")
}

// GetNavSummary returns message and system message summary for nav bar.
// GET /api/message/nav/system
func (h *MessageHandler) GetNavSummary(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	result, appErr := h.messageService.GetNavSummary(c.Context(), user.UID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, result)
}

// MarkAllRead marks all user notification messages as read.
// PUT /api/message/system/read
func (h *MessageHandler) MarkAllRead(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.messageService.MarkAllRead(c.Context(), user.UID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "已标记全部已读")
}

// GetNavContact returns chat room list for message sidebar.
// GET /api/message/nav/contact
func (h *MessageHandler) GetNavContact(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	uid := user.UID

	type roomRow struct {
		ID                  int     `gorm:"column:id"`
		Name                string  `gorm:"column:name"`
		Avatar              string  `gorm:"column:avatar"`
		Type                string  `gorm:"column:type"`
		LastMessageContent  string  `gorm:"column:last_message_content"`
		LastMessageTime     *string `gorm:"column:last_message_time"`
	}
	var rooms []roomRow
	if err := h.db.Table("chat_room cr").
		Select(`cr.id, cr.name, cr.avatar, cr.type,
			cr.last_message_content, cr.last_message_time`).
		Joins("JOIN chat_room_participant crp ON crp.chat_room_id = cr.id").
		Where("crp.user_id = ? AND cr.last_message_sender_id != 0 AND cr.last_message_time IS NOT NULL", uid).
		Order("cr.last_message_time DESC").
		Scan(&rooms).Error; err != nil {
		return response.Error(c, errors.ErrInternal("查询聊天室失败"))
	}

	if len(rooms) == 0 {
		return response.OK(c, []fiber.Map{})
	}

	roomIDs := make([]int, len(rooms))
	for i, r := range rooms {
		roomIDs[i] = r.ID
	}

	// Participants
	type participantRow struct {
		ChatRoomID int    `gorm:"column:chat_room_id"`
		UserID     int    `gorm:"column:user_id"`
		UserName   string `gorm:"column:user_name"`
		UserAvatar string `gorm:"column:user_avatar"`
	}
	var participants []participantRow
	h.db.Table("chat_room_participant p").
		Select("p.chat_room_id, p.user_id, u.name AS user_name, u.avatar AS user_avatar").
		Joins(`LEFT JOIN "user" u ON u.id = p.user_id`).
		Where("p.chat_room_id IN ?", roomIDs).
		Scan(&participants)
	roomParts := make(map[int][]participantRow)
	for _, p := range participants {
		roomParts[p.ChatRoomID] = append(roomParts[p.ChatRoomID], p)
	}

	// Unread counts
	type countRow struct {
		ChatRoomID int `gorm:"column:chat_room_id"`
		Count      int `gorm:"column:count"`
	}
	var unreads []countRow
	h.db.Table("chat_message cm").
		Select("cm.chat_room_id, COUNT(*) AS count").
		Where("cm.chat_room_id IN ? AND cm.sender_id != ?", roomIDs, uid).
		Where("cm.id NOT IN (SELECT chat_message_id FROM chat_message_read_by WHERE user_id = ?)", uid).
		Group("cm.chat_room_id").
		Scan(&unreads)
	unreadMap := make(map[int]int)
	for _, u := range unreads {
		unreadMap[u.ChatRoomID] = u.Count
	}

	// Total message counts
	var totals []countRow
	h.db.Table("chat_message").
		Select("chat_room_id, COUNT(*) AS count").
		Where("chat_room_id IN ?", roomIDs).
		Group("chat_room_id").
		Scan(&totals)
	totalMap := make(map[int]int)
	for _, t := range totals {
		totalMap[t.ChatRoomID] = t.Count
	}

	items := make([]fiber.Map, len(rooms))
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
		items[i] = fiber.Map{
			"chatroomName":    r.Name,
			"content":         r.LastMessageContent,
			"lastMessageTime": r.LastMessageTime,
			"count":           totalMap[r.ID],
			"unreadCount":     unreadMap[r.ID],
			"route":           route,
			"title":           title,
			"avatar":          avatar,
		}
	}

	return response.OK(c, items)
}

func generateRoomID(uid1, uid2 int) string {
	if uid1 < uid2 {
		return fmt.Sprintf("%d-%d", uid1, uid2)
	}
	return fmt.Sprintf("%d-%d", uid2, uid1)
}

// findOrCreatePrivateRoom finds an existing private chat room between two users
// by checking participants (not room name, which may be stale after migration).
// Creates a new room if none exists.
func (h *MessageHandler) findOrCreatePrivateRoom(uid1, uid2 int) (int, string) {
	// Find room where both users are participants
	type roomRow struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var room roomRow
	h.db.Raw(`
		SELECT cr.id, cr.name FROM chat_room cr
		WHERE cr.type = 'private'
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		LIMIT 1`, uid1, uid2).Scan(&room)

	if room.ID > 0 {
		return room.ID, room.Name
	}

	// Create new room
	roomName := generateRoomID(uid1, uid2)
	h.db.Exec(
		`INSERT INTO chat_room (name, type, created, updated) VALUES (?, 'private', NOW(), NOW())`,
		roomName,
	)
	h.db.Raw(`SELECT id, name FROM chat_room WHERE name = ?`, roomName).Scan(&room)
	if room.ID > 0 {
		h.db.Exec(
			`INSERT INTO chat_room_participant (chat_room_id, user_id, created, updated) VALUES (?, ?, NOW(), NOW()), (?, ?, NOW(), NOW())`,
			room.ID, uid1, room.ID, uid2,
		)
	}
	return room.ID, roomName
}

// GetChatHistory returns chat message history.
// GET /api/message/chat/history
func (h *MessageHandler) GetChatHistory(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ReceiverUID int `query:"receiverUid" validate:"required,min=1"`
		Page        int `query:"page" validate:"min=1"`
		Limit       int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	uid := user.UID
	if req.ReceiverUID == uid {
		return response.Error(c, errors.ErrBadRequest("不能给自己发送消息"))
	}

	roomID, roomName := h.findOrCreatePrivateRoom(uid, req.ReceiverUID)
	if roomID == 0 {
		return response.OK(c, []fiber.Map{})
	}

	// Fetch messages by room_id (reliable) or room name (for old data)
	type msgRow struct {
		ID           int     `gorm:"column:id"`
		ChatroomName string  `gorm:"column:chatroom_name"`
		SenderID     int     `gorm:"column:sender_id"`
		SenderName   string  `gorm:"column:sender_name"`
		SenderAvatar string  `gorm:"column:sender_avatar"`
		ReceiverID   int     `gorm:"column:receiver_id"`
		Content      string  `gorm:"column:content"`
		IsRecall     bool    `gorm:"column:is_recall"`
		Created      string  `gorm:"column:created"`
		RecallTime   *string `gorm:"column:recall_time"`
		EditTime     *string `gorm:"column:edit_time"`
	}
	var msgs []msgRow
	offset := (req.Page - 1) * req.Limit
	h.db.Table("chat_message cm").
		Select(`cm.id, cm.chatroom_name, cm.sender_id,
			u.name AS sender_name, u.avatar AS sender_avatar,
			cm.receiver_id, cm.content, cm.is_recall,
			cm.created, cm.recall_time, cm.edit_time`).
		Joins(`LEFT JOIN "user" u ON u.id = cm.sender_id`).
		Where("cm.chat_room_id = ? OR cm.chatroom_name = ?", roomID, roomName).
		Order("cm.id DESC").
		Offset(offset).Limit(req.Limit).
		Scan(&msgs)

	// Mark as read
	if len(msgs) > 0 {
		msgIDs := make([]int, 0, len(msgs))
		for _, m := range msgs {
			if m.SenderID != uid {
				msgIDs = append(msgIDs, m.ID)
			}
		}
		if len(msgIDs) > 0 {
			for _, mid := range msgIDs {
				h.db.Exec(
					`INSERT INTO chat_message_read_by (chat_message_id, user_id, created, updated) VALUES (?, ?, NOW(), NOW()) ON CONFLICT DO NOTHING`,
					mid, uid,
				)
			}
		}
	}

	// Build response (reverse to chronological order)
	messages := make([]fiber.Map, len(msgs))
	for i, m := range msgs {
		messages[len(msgs)-1-i] = fiber.Map{
			"id":           m.ID,
			"chatroomName": m.ChatroomName,
			"sender": fiber.Map{
				"id": m.SenderID, "name": m.SenderName, "avatar": m.SenderAvatar,
			},
			"receiverUid": m.ReceiverID,
			"content":     m.Content,
			"isRecall":    m.IsRecall,
			"created":     m.Created,
			"recallTime":  m.RecallTime,
			"editTime":    m.EditTime,
			"readBy":      []fiber.Map{},
		}
	}

	return response.OK(c, messages)
}

// SendChatMessage sends a chat message (replaces socket.io).
// POST /api/message/chat/send
func (h *MessageHandler) SendChatMessage(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ReceiverUID int    `json:"receiverUid" validate:"required,min=1"`
		Content     string `json:"content" validate:"required,min=1,max=1007"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	uid := user.UID
	if req.ReceiverUID == uid {
		return response.Error(c, errors.ErrBadRequest("不能给自己发送消息"))
	}

	roomID, roomName := h.findOrCreatePrivateRoom(uid, req.ReceiverUID)
	if roomID == 0 {
		return response.Error(c, errors.ErrInternal("创建聊天室失败"))
	}

	now := time.Now()
	h.db.Exec(
		`INSERT INTO chat_message (chat_room_id, chatroom_name, sender_id, receiver_id, content, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		roomID, roomName, uid, req.ReceiverUID, req.Content, now, now,
	)

	h.db.Exec(
		`UPDATE chat_room SET last_message_content = ?, last_message_time = ?,
		last_message_sender_id = ?, last_message_sender_name = ?, updated = ? WHERE id = ?`,
		req.Content, now, uid, user.Name, now, roomID,
	)

	return response.OKMessage(c, "发送成功")
}
