package service

import (
	"fmt"

	msgModel "kun-galgame-api/internal/message/model"
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

// InteractionHelpers encapsulates the cross-cutting side effects used by topic
// interactions: moemoepoint adjustment and de-duplicated messaging.
//
// All methods operate on a caller-supplied *gorm.DB (typically a running
// transaction) so the caller controls atomicity.
type InteractionHelpers struct{}

// AdjustMoemoepoint adds `delta` to the target user's moemoepoint in the
// kungal_user_state table (the OAuth-migration successor to the deleted
// user.moemoepoint column). No-op when userID<=0 or delta==0.
func (InteractionHelpers) AdjustMoemoepoint(tx *gorm.DB, userID int, delta int) {
	if userID <= 0 || delta == 0 {
		return
	}
	tx.Model(&userModel.KungalUserState{}).Where("user_id = ?", userID).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", delta))
}

// CreateTopicMessage creates a notification for a topic-level action
// (liked/upvoted/favorite). Dedup is performed on (sender, receiver, type, link).
func (InteractionHelpers) CreateTopicMessage(tx *gorm.DB, senderID, receiverID int, msgType string, topicID int) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/topic/%d", topicID)
	createDedupMessage(tx, senderID, receiverID, msgType, "", link)
}

// CreateTopicMessageWithContent is like CreateTopicMessage but stores a
// content preview (used for reply/comment like notifications).
func (InteractionHelpers) CreateTopicMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/topic/%d", topicID)
	createDedupMessage(tx, senderID, receiverID, msgType, content, link)
}

// CreateReplyMessage creates a (non-deduped) reply/commented notification for
// every target/owner separately. Callers should ensure this is only invoked
// once per distinct (sender,receiver).
func (InteractionHelpers) CreateReplyMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/topic/%d", topicID)
	tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
	})
}

// createDedupMessage creates a Message only if no row already exists with the
// same (sender, receiver, type, link) triple. Same-user actions are a no-op.
func createDedupMessage(tx *gorm.DB, senderID, receiverID int, msgType, content, link string) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}
	tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
	})
}

// truncate trims `s` to `maxLen` runes (rune-aware, avoids splitting multibyte chars).
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
