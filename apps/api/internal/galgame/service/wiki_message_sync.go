package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	userRepo "kun-galgame-api/internal/user/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// WikiMessageSync drives the periodic ingestion of admin-triggered events
// (approved / declined / banned / unbanned) from wiki's /messages/feed.
// See docs/galgame_wiki/07-submission.md §调用方 cron 同步本地 status for
// the design rationale.
//
// Side effects per type (matches the kungal/wiki integration decision matrix):
//
//	approved    +3 moemoepoint to TargetUserID (the submitter).
//	            Idempotent via Redis SETNX guard so a re-run of the same
//	            cursor doesn't double-award.
//	            Also creates the local galgame stub — approved is the moment
//	            a galgame becomes publicly visible and needs a row to
//	            anchor the list query.
//	banned      Delete the local galgame stub (CASCADE cleans interactions).
//	            Lazy-loaded stubs from prior likes/comments become orphan
//	            references after a ban; this cron is what tidies them up.
//	declined    No kungal-side action (the user gets a /messages/mine
//	            notification and can PATCH to re-submit).
//	unbanned    No kungal-side moemoe re-grant (the +3 was given at the
//	            original approval and would double-count on
//	            ban-then-unban).
//
// galgame=null (hard-deleted) → treated like banned (DELETE stub).
type WikiMessageSync struct {
	wikiClient  *client.GalgameClient
	galgameRepo *repository.GalgameRepository
	stateRepo   *userRepo.StateRepository
	rdb         *redis.Client
}

func NewWikiMessageSync(
	wikiClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	stateRepo *userRepo.StateRepository,
	rdb *redis.Client,
) *WikiMessageSync {
	return &WikiMessageSync{
		wikiClient:  wikiClient,
		galgameRepo: galgameRepo,
		stateRepo:   stateRepo,
		rdb:         rdb,
	}
}

const (
	// cronCursorKey is the Redis key holding the last-processed message id.
	// Single global cursor (server-wide), so storing in Redis rather than
	// a kungal SQL row keeps the cron stateless across restarts.
	cronCursorKey = "wiki:msg:cron:since"

	// processedGuardTTL is the per-message idempotency window. 30 days is
	// far larger than the cursor would ever realistically rewind, but
	// cheap (small Redis keys, no scan).
	processedGuardTTL = 30 * 24 * time.Hour

	// messagesFeedBatch is the max number of messages requested per page.
	// Matches the wiki's stated 1000 limit.
	messagesFeedBatch = 1000

	// maxPagesPerRun guards against a runaway feed (wiki misbehaving and
	// always returning has_more=true). The cursor still advances within
	// the cap, so the next run picks up where this one stopped.
	maxPagesPerRun = 50
)

// Run executes one sync cycle. Safe to call concurrently — events use
// SETNX guards so duplicate processing only re-writes the cursor.
// Designed to be cheap when there's nothing new (one HTTP GET that
// returns has_more=false and items=[]).
func (s *WikiMessageSync) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sinceID := s.readCursor(ctx)
	maxSeen := sinceID
	startedFrom := sinceID
	pages := 0

	for {
		pages++
		if pages > maxPagesPerRun {
			slog.Warn("wiki message sync 翻页超过上限, 中断本轮",
				"pages", pages, "since_id", maxSeen)
			break
		}

		feed, appErr := s.wikiClient.MessagesFeed(ctx, sinceID, messagesFeedBatch)
		if appErr != nil {
			slog.Warn("wiki message feed 拉取失败", "error", appErr, "since_id", sinceID)
			return
		}

		for _, msg := range feed.Items {
			s.applyMessage(ctx, msg)
			if msg.ID > maxSeen {
				maxSeen = msg.ID
			}
		}

		if !feed.HasMore || len(feed.Items) == 0 {
			break
		}
		sinceID = maxSeen
	}

	if maxSeen > startedFrom {
		s.writeCursor(ctx, maxSeen)
		slog.Info("wiki 消息同步完成",
			"from", startedFrom, "to", maxSeen, "pages", pages)
	}
}

func (s *WikiMessageSync) applyMessage(ctx context.Context, msg client.WikiMessage) {
	// Per-message idempotency. Concurrent runs of this cron (e.g. cron tick
	// during a restart) would otherwise double-award moemoe.
	guardKey := "wiki:msg:processed:" + strconv.FormatInt(msg.ID, 10)
	acquired, err := s.rdb.SetNX(ctx, guardKey, "1", processedGuardTTL).Result()
	if err != nil {
		slog.Warn("处理 wiki 消息时获取去重锁失败",
			"msg_id", msg.ID, "error", err)
		return
	}
	if !acquired {
		return
	}

	// Hard-deleted galgame: clean any orphan stub.
	if msg.Galgame == nil {
		s.galgameRepo.DeleteLocalStub(msg.GalgameID)
		return
	}

	switch msg.Type {
	case "approved":
		// approved is the moment a galgame becomes publicly visible —
		// seed the kungal stub here so the list query has something to
		// anchor on. Award the submitter +3 atomically in the same tx
		// so a partial failure rolls back both.
		if msg.TargetUserID == nil || *msg.TargetUserID <= 0 {
			slog.Warn("approved 消息缺少 target_user_id, 仅创建 stub 跳过奖励",
				"msg_id", msg.ID, "gid", msg.GalgameID)
			s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
				s.galgameRepo.CreateLocalStub(tx, msg.GalgameID)
				return nil
			})
			return
		}
		target := *msg.TargetUserID
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.galgameRepo.CreateLocalStub(tx, msg.GalgameID)
			return s.stateRepo.AdjustMoemoepointTx(tx, target, constants.RewardCreateGalgame)
		})

	case "banned":
		s.galgameRepo.DeleteLocalStub(msg.GalgameID)

	case "declined", "unbanned":
		// No kungal-side action — see type-doc above.
	default:
		slog.Warn("收到未识别的 wiki 消息类型, 跳过",
			"type", msg.Type, "msg_id", msg.ID)
	}
}

func (s *WikiMessageSync) readCursor(ctx context.Context) int64 {
	v, err := s.rdb.Get(ctx, cronCursorKey).Int64()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		slog.Warn("读取 wiki 消息游标失败 (从 0 开始)", "error", err)
		return 0
	}
	return v
}

func (s *WikiMessageSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, cronCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入 wiki 消息游标失败", "id", id, "error", err)
	}
}
