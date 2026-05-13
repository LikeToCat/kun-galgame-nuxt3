package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
)

// WikiMessageService exposes user-facing reads of the wiki message stream
// (GET /messages/mine), the admin queue (GET /admin/galgame/messages), and
// the kungal-local read-state cursor. The actual messages are owned by
// wiki — kungal proxies them through with the user's Bearer attached.
type WikiMessageService struct {
	wikiClient *client.GalgameClient
	repo       *repository.WikiMessageRepository
}

func NewWikiMessageService(
	wikiClient *client.GalgameClient,
	repo *repository.WikiMessageRepository,
) *WikiMessageService {
	return &WikiMessageService{wikiClient: wikiClient, repo: repo}
}

// MessagesMine proxies the user's notification feed from wiki. wiki resolves
// the Bearer's JWT.uid and returns messages where target_user_id matches.
func (s *WikiMessageService) MessagesMine(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.GetWithToken(ctx, "/galgame/messages/mine", token, query)
}

// AdminMessages proxies the admin moderation queue. The handler must have
// already checked RequireRole(2) — this is a thin pass-through and does
// not re-validate role.
func (s *WikiMessageService) AdminMessages(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.GetWithToken(ctx, "/admin/galgame/messages", token, query)
}

// GetReadState returns the user's last-read marker (0 if never set).
func (s *WikiMessageService) GetReadState(userID int) (int64, error) {
	row, err := s.repo.FindOrZero(userID)
	if err != nil {
		return 0, err
	}
	return row.LastReadMessageID, nil
}

// SetReadState advances the user's last-read marker. The repo applies a
// GREATEST() so concurrent calls / stale tabs can't rewind it.
func (s *WikiMessageService) SetReadState(userID int, lastReadID int64) error {
	return s.repo.UpsertForward(userID, lastReadID)
}
