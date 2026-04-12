package repository

import (
	"kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Message CRUD
// ──────────────────────────────────────────

type MessageRow struct {
	ID             int
	SenderID       int
	SenderName     string
	SenderAvatar   string
	ReceiverID     int
	Link           string
	Content        string
	Status         string
	Type           string
	CreatedAt      string
}

func (r *MessageRepository) FindMessages(
	receiverID int,
	msgType, sortOrder string,
	page, limit int,
) ([]MessageRow, int64, error) {
	var rows []MessageRow
	var total int64

	query := r.db.Table("message m").
		Select(`m.id, m.sender_id, u.name AS sender_name, u.avatar AS sender_avatar,
			m.receiver_id, m.link, m.content, m.status, m.type, m.created AS created_at`).
		Joins(`LEFT JOIN "user" u ON u.id = m.sender_id`).
		Where("m.receiver_id = ?", receiverID)

	if msgType != "" {
		query = query.Where("m.type = ?", msgType)
	}

	query.Count(&total)

	err := query.
		Order("m.created " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error

	return rows, total, err
}

func (r *MessageRepository) DeleteByIDAndReceiver(id, receiverID int) error {
	return r.db.Where("id = ? AND receiver_id = ?", id, receiverID).
		Delete(&model.Message{}).Error
}

func (r *MessageRepository) MarkAllRead(receiverID int) error {
	return r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND status = 'unread'", receiverID).
		Update("status", "read").Error
}

// ──────────────────────────────────────────
// System messages
// ──────────────────────────────────────────

type SystemMessageRow struct {
	ID          int
	Status      string
	ContentEnUS string
	ContentJaJP string
	ContentZhCN string
	ContentZhTW string
	UserID      int
	UserName    string
	UserAvatar  string
	CreatedAt   string
}

func (r *MessageRepository) FindSystemMessages() ([]SystemMessageRow, error) {
	var rows []SystemMessageRow
	err := r.db.Table("system_message sm").
		Select(`sm.id, sm.status, sm.content_en_us, sm.content_ja_jp,
			sm.content_zh_cn, sm.content_zh_tw,
			sm.user_id, u.name AS user_name, u.avatar AS user_avatar,
			sm.created AS created_at`).
		Joins(`LEFT JOIN "user" u ON u.id = sm.user_id`).
		Order("sm.created DESC").
		Find(&rows).Error
	return rows, err
}

func (r *MessageRepository) MarkAllSystemRead() error {
	return r.db.Model(&model.SystemMessage{}).
		Where("status = 'unread'").
		Update("status", "read").Error
}

// ──────────────────────────────────────────
// Nav summary
// ──────────────────────────────────────────

func (r *MessageRepository) GetNavSummary(uid int) ([]map[string]any, error) {
	// Notice messages
	var noticeTotal, noticeUnread int64
	r.db.Model(&model.Message{}).Where("receiver_id = ?", uid).Count(&noticeTotal)
	r.db.Model(&model.Message{}).Where("receiver_id = ? AND status = 'unread'", uid).Count(&noticeUnread)

	var latestNotice model.Message
	r.db.Where("receiver_id = ?", uid).Order("created DESC").First(&latestNotice)

	// System messages
	var sysTotal, sysUnread int64
	r.db.Model(&model.SystemMessage{}).Count(&sysTotal)
	r.db.Model(&model.SystemMessage{}).Where("status = 'unread'").Count(&sysUnread)

	var latestSys model.SystemMessage
	r.db.Order("created DESC").First(&latestSys)

	result := []map[string]any{
		{
			"type":          "notice",
			"totalCount":    noticeTotal,
			"unreadCount":   noticeUnread,
			"latestContent": latestNotice.Content,
			"latestTime":    latestNotice.CreatedAt,
		},
		{
			"type":          "system",
			"totalCount":    sysTotal,
			"unreadCount":   sysUnread,
			"latestContent": latestSys.ContentZhCN,
			"latestTime":    latestSys.CreatedAt,
		},
	}
	return result, nil
}
