package repository

import (
	stderrors "errors"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// WikiMessageRepository owns the kungal-local wiki_message_read_state table.
// The wiki messages themselves live in the wiki service; we just track each
// user's "read up to" cursor so the frontend can compute unread counts.
type WikiMessageRepository struct {
	db *gorm.DB
}

func NewWikiMessageRepository(db *gorm.DB) *WikiMessageRepository {
	return &WikiMessageRepository{db: db}
}

// FindOrZero returns the user's read-state row, or a zero-valued struct
// (UserID set, LastReadMessageID=0) when the user has never marked any
// message as read.
func (r *WikiMessageRepository) FindOrZero(userID int) (*model.WikiMessageReadState, error) {
	var row model.WikiMessageReadState
	err := r.db.First(&row, "user_id = ?", userID).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return &model.WikiMessageReadState{UserID: userID}, nil
	}
	return &row, err
}

// UpsertForward writes the user's last-read marker. Only moves forward —
// stale requests trying to lower the marker are silently no-op'd, which
// avoids "I marked everything read on tab A, tab B is still mid-load with
// an older message list and tries to rewind the marker" races.
//
// Implemented as raw INSERT...ON CONFLICT to use Postgres's GREATEST() —
// expressing this through GORM's clause builder is uglier than just
// writing the SQL.
func (r *WikiMessageRepository) UpsertForward(userID int, lastReadID int64) error {
	return r.db.Exec(`
		INSERT INTO wiki_message_read_state (user_id, last_read_message_id, updated_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET last_read_message_id = GREATEST(
		        wiki_message_read_state.last_read_message_id,
		        EXCLUDED.last_read_message_id
		    ),
		    updated_at = NOW()
	`, userID, lastReadID).Error
}
