package repository

import (
	"time"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Row types
// ──────────────────────────────────────────

// RoomListRow is a chat room entry for the contacts sidebar.
type RoomListRow struct {
	ID                 int     `gorm:"column:id"`
	Name               string  `gorm:"column:name"`
	Avatar             string  `gorm:"column:avatar"`
	Type               string  `gorm:"column:type"`
	LastMessageContent string  `gorm:"column:last_message_content"`
	LastMessageTime    *string `gorm:"column:last_message_time"`
}

// ParticipantRow is a row from chat_room_participant joined with user.
type ParticipantRow struct {
	ChatRoomID int    `gorm:"column:chat_room_id"`
	UserID     int    `gorm:"column:user_id"`
	UserName   string `gorm:"column:user_name"`
	UserAvatar string `gorm:"column:user_avatar"`
}

// CountRow holds a per-room count (unread or total).
type CountRow struct {
	ChatRoomID int `gorm:"column:chat_room_id"`
	Count      int `gorm:"column:count"`
}

// RoomRef is a minimal room reference (id + name).
type RoomRef struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// ChatMessageRow is a joined chat_message row with sender user info.
type ChatMessageRow struct {
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

// ──────────────────────────────────────────
// Room / participant queries
// ──────────────────────────────────────────

// FindRoomsForUser returns every chat room the user participates in that has
// at least one message, ordered by last message time DESC.
func (r *ChatRepository) FindRoomsForUser(uid int) ([]RoomListRow, error) {
	var rooms []RoomListRow
	err := r.db.Table("chat_room cr").
		Select(`cr.id, cr.name, cr.avatar, cr.type,
			cr.last_message_content, cr.last_message_time`).
		Joins("JOIN chat_room_participant crp ON crp.chat_room_id = cr.id").
		Where("crp.user_id = ? AND cr.last_message_sender_id != 0 AND cr.last_message_time IS NOT NULL", uid).
		Order("cr.last_message_time DESC").
		Scan(&rooms).Error
	return rooms, err
}

// FindParticipantsByRoomIDs returns all participants for the given room IDs,
// each joined with user name + avatar.
func (r *ChatRepository) FindParticipantsByRoomIDs(roomIDs []int) []ParticipantRow {
	var rows []ParticipantRow
	r.db.Table("chat_room_participant p").
		Select("p.chat_room_id, p.user_id, u.name AS user_name, u.avatar AS user_avatar").
		Joins(`LEFT JOIN "user" u ON u.id = p.user_id`).
		Where("p.chat_room_id IN ?", roomIDs).
		Scan(&rows)
	return rows
}

// CountUnreadByRoomIDs returns unread-message counts (per room) for the given user:
// messages in the room NOT sent by the user AND not present in chat_message_read_by.
func (r *ChatRepository) CountUnreadByRoomIDs(roomIDs []int, uid int) []CountRow {
	var rows []CountRow
	r.db.Table("chat_message cm").
		Select("cm.chat_room_id, COUNT(*) AS count").
		Where("cm.chat_room_id IN ? AND cm.sender_id != ?", roomIDs, uid).
		Where("cm.id NOT IN (SELECT chat_message_id FROM chat_message_read_by WHERE user_id = ?)", uid).
		Group("cm.chat_room_id").
		Scan(&rows)
	return rows
}

// CountTotalByRoomIDs returns total-message counts per room.
func (r *ChatRepository) CountTotalByRoomIDs(roomIDs []int) []CountRow {
	var rows []CountRow
	r.db.Table("chat_message").
		Select("chat_room_id, COUNT(*) AS count").
		Where("chat_room_id IN ?", roomIDs).
		Group("chat_room_id").
		Scan(&rows)
	return rows
}

// FindPrivateRoomBetween looks up the existing private chat room between two
// users by checking the participant table (NOT by room name — names may be
// stale after OAuth migration changed user IDs). Returns the zero value if
// no room exists.
func (r *ChatRepository) FindPrivateRoomBetween(uid1, uid2 int) RoomRef {
	var room RoomRef
	r.db.Raw(`
		SELECT cr.id, cr.name FROM chat_room cr
		WHERE cr.type = 'private'
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		LIMIT 1`, uid1, uid2).Scan(&room)
	return room
}

// CreatePrivateRoom inserts a new private chat room with both users as
// participants. Returns the new room (id + name); id will be 0 if creation
// failed.
func (r *ChatRepository) CreatePrivateRoom(roomName string, uid1, uid2 int) RoomRef {
	r.db.Exec(
		`INSERT INTO chat_room (name, type, created, updated) VALUES (?, 'private', NOW(), NOW())`,
		roomName,
	)
	var room RoomRef
	r.db.Raw(`SELECT id, name FROM chat_room WHERE name = ?`, roomName).Scan(&room)
	if room.ID > 0 {
		r.db.Exec(
			`INSERT INTO chat_room_participant (chat_room_id, user_id, created, updated) VALUES (?, ?, NOW(), NOW()), (?, ?, NOW(), NOW())`,
			room.ID, uid1, room.ID, uid2,
		)
	}
	return room
}

// ──────────────────────────────────────────
// Chat message queries
// ──────────────────────────────────────────

// FindMessagesByRoom returns chat messages for a room, ordered by id DESC
// (newest first), joined with sender user info. Matches by chat_room_id
// OR legacy chatroom_name (for old data predating the migration).
func (r *ChatRepository) FindMessagesByRoom(roomID int, roomName string, page, limit int) []ChatMessageRow {
	var rows []ChatMessageRow
	offset := (page - 1) * limit
	r.db.Table("chat_message cm").
		Select(`cm.id, cm.chatroom_name, cm.sender_id,
			u.name AS sender_name, u.avatar AS sender_avatar,
			cm.receiver_id, cm.content, cm.is_recall,
			cm.created, cm.recall_time, cm.edit_time`).
		Joins(`LEFT JOIN "user" u ON u.id = cm.sender_id`).
		Where("cm.chat_room_id = ? OR cm.chatroom_name = ?", roomID, roomName).
		Order("cm.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows)
	return rows
}

// MarkMessagesRead inserts (chat_message_id, user_id) rows into
// chat_message_read_by, ignoring duplicates. A no-op if msgIDs is empty.
func (r *ChatRepository) MarkMessagesRead(msgIDs []int, uid int) {
	for _, mid := range msgIDs {
		r.db.Exec(
			`INSERT INTO chat_message_read_by (chat_message_id, user_id, created, updated) VALUES (?, ?, NOW(), NOW()) ON CONFLICT DO NOTHING`,
			mid, uid,
		)
	}
}

// InsertChatMessage writes a new chat message to chat_message.
func (r *ChatRepository) InsertChatMessage(roomID int, roomName string, senderID, receiverID int, content string, now time.Time) {
	r.db.Exec(
		`INSERT INTO chat_message (chat_room_id, chatroom_name, sender_id, receiver_id, content, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		roomID, roomName, senderID, receiverID, content, now, now,
	)
}

// UpdateRoomLastMessage refreshes chat_room.last_message_* fields.
func (r *ChatRepository) UpdateRoomLastMessage(roomID int, content string, senderID int, senderName string, now time.Time) {
	r.db.Exec(
		`UPDATE chat_room SET last_message_content = ?, last_message_time = ?,
		last_message_sender_id = ?, last_message_sender_name = ?, updated = ? WHERE id = ?`,
		content, now, senderID, senderName, now, roomID,
	)
}
