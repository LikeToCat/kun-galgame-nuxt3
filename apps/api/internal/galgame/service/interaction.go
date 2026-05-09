package service

import (
	"fmt"

	msgModel "kun-galgame-api/internal/message/model"
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

// InteractionHelpers encapsulates the two cross-cutting side effects used by
// galgame interactions: moemoepoint adjustment and de-duplicated messaging.
//
// It operates on a caller-supplied *gorm.DB (typically a running transaction)
// so the caller controls atomicity.
type InteractionHelpers struct{}

// AdjustMoemoepoint adds `delta` to the target user's moemoepoint in the
// kungal_user_state table (the post-OAuth-migration source of truth). The
// legacy user.moemoepoint column is no longer authoritative.
func (InteractionHelpers) AdjustMoemoepoint(tx *gorm.DB, userID int, delta int) {
	if userID <= 0 || delta == 0 {
		return
	}
	tx.Model(&userModel.KungalUserState{}).Where("user_id = ?", userID).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", delta))
}

// CreateGalgameMessage creates a notification from `senderID` → `receiverID`
// for a galgame-related action, deduplicating against any existing row with
// the same (sender, receiver, type, link) triple. Same-user actions are a no-op.
func (h InteractionHelpers) CreateGalgameMessage(tx *gorm.DB, senderID, receiverID int, msgType string, galgameID int) {
	h.CreateGalgameMessageWithContent(tx, senderID, receiverID, msgType, "", galgameID)
}

// CreateGalgameMessageWithContent is the variant that records a content
// preview (used by resource liked/expired notifications). Dedup applies on
// (sender, receiver, type, link) — content is informational only.
func (InteractionHelpers) CreateGalgameMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	galgameID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/galgame/%d", galgameID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}

	tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: content, Link: link, Status: "unread",
	})
}

