package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

// SubmissionService handles the user-driven submission lifecycle for new
// galgames per docs/galgame_wiki/07-submission.md.
//
// Reward policy (decision 1 in the kungal/wiki integration plan):
//
//	submit → 0 (no reward; deferred until approval to deter spam)
//	claim  → +3 (claim publishes immediately, no review queue)
//	approved (via cron) → +3 (handled in WikiMessageSync, not here)
//	declined / delete → no reward to reclaim (never granted)
type SubmissionService struct {
	wikiClient  *client.GalgameClient
	galgameRepo *repository.GalgameRepository
	stateRepo   *userRepo.StateRepository
}

func NewSubmissionService(
	wikiClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	stateRepo *userRepo.StateRepository,
) *SubmissionService {
	return &SubmissionService{
		wikiClient:  wikiClient,
		galgameRepo: galgameRepo,
		stateRepo:   stateRepo,
	}
}

// Submit forwards a pending submission to wiki. NO local side effects:
//   - no stub (status=3 isn't publicly listable, so no need to seed the
//     local `galgame` index; if the user self-interacts later the
//     interaction path lazy-creates the stub)
//   - no moemoepoint (deferred to approval via cron)
//
// Wiki enforces the daily quota (20009) and vndb_id format/uniqueness.
func (s *SubmissionService) Submit(
	ctx context.Context,
	token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.SubmitDraft(ctx, token, body, contentType)
}

// Claim flips a VNDB-source draft (status=2) directly to published
// (status=0) on the wiki, then in a separate local transaction:
//   - creates the local stub so the galgame appears in kungal's list query;
//   - awards the claimer +3 moemoepoint, atomic with a FOR UPDATE lock on
//     kungal_user_state so concurrent claims by the same user can't both
//     pass through (the wiki side also makes claim idempotent at the row
//     level by checking status=2).
//
// Wiki failure → no local side effect runs (correct).
// Local failure after wiki success → log; the +3 is forfeited but the
// publish itself stands. Avoiding a half-rollback of wiki state.
func (s *SubmissionService) Claim(
	ctx context.Context,
	userID int,
	token string,
	gid int,
) (json.RawMessage, *errors.AppError) {
	data, appErr := s.wikiClient.ClaimDraft(ctx, token, gid)
	if appErr != nil {
		return nil, appErr
	}

	txErr := s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
		if _, lockErr := s.stateRepo.LockForUpdate(tx, userID); lockErr != nil {
			return lockErr
		}
		s.galgameRepo.CreateLocalStub(tx, gid)
		return s.stateRepo.AdjustMoemoepointTx(tx, userID, constants.RewardCreateGalgame)
	})
	if txErr != nil {
		slog.Warn("claim 本地副作用失败 (wiki 已成功)",
			"gid", gid, "uid", userID, "error", txErr)
	}
	return data, nil
}

// PatchDraft forwards an edit to the user's own pending/declined draft.
// No local side effects — wiki holds all metadata and the row remains
// invisible to other users until approval.
func (s *SubmissionService) PatchDraft(
	ctx context.Context,
	token string,
	gid int,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.PatchDraft(ctx, token, gid, body, contentType)
}

// DeleteDraft hard-deletes the user's own pending/declined draft on the
// wiki, then defensively removes any kungal stub the user may have
// lazy-created by self-interacting with the pending row. Idempotent.
//
// Wiki failure → don't touch local state.
func (s *SubmissionService) DeleteDraft(
	ctx context.Context,
	token string,
	gid int,
) *errors.AppError {
	if appErr := s.wikiClient.DeleteDraft(ctx, token, gid); appErr != nil {
		return appErr
	}
	s.galgameRepo.DeleteLocalStub(gid)
	return nil
}

// ListMine proxies GET /galgame/mine — the user's own submissions across
// all statuses. Thin pass-through; identity comes from the forwarded Bearer.
func (s *SubmissionService) ListMine(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.GetWithToken(ctx, "/galgame/mine", token, query)
}

// SearchWithPending proxies GET /galgame/search?include_pending=true — the
// "发布向导" flow. The handler is expected to have already set the
// include_pending flag; we forward as-is so wiki sees the caller's JWT.uid
// and merges the caller's pending hits into the response.
func (s *SubmissionService) SearchWithPending(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.GetWithToken(ctx, "/galgame/search", token, query)
}
