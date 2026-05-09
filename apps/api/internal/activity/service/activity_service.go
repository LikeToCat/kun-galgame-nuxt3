package service

import (
	"context"
	"fmt"

	"kun-galgame-api/internal/activity/dto"
	"kun-galgame-api/internal/activity/repository"
	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type ActivityService struct {
	repo       *repository.ActivityRepository
	wikiGC     *galgameClient.GalgameClient
	userClient *userclient.Client
}

func NewActivityService(
	repo *repository.ActivityRepository,
	gc *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *ActivityService {
	return &ActivityService{repo: repo, wikiGC: gc, userClient: userClient}
}

// Result holds a paginated activity list.
type Result struct {
	Items []dto.ActivityItem
	Total int64
}

// GetActivity returns a filtered activity feed. If the type is "all",
// it falls back to GetTimeline.
func (s *ActivityService) GetActivity(ctx context.Context, typeStr string, page, limit int) (*Result, *errors.AppError) {
	if typeStr == "all" {
		return s.GetTimeline(ctx, page, limit)
	}

	src, ok := s.repo.GetSource(typeStr)
	if !ok {
		return &Result{Items: []dto.ActivityItem{}, Total: 0}, nil
	}

	rows, total, err := s.repo.FetchSingleSource(src, page, limit)
	if err != nil {
		return nil, errors.ErrInternal("查询活动数据失败")
	}
	items := rowsToItems(rows)
	s.enrichGalgameItems(ctx, rows, items)
	s.hydrateActors(ctx, items)
	return &Result{Items: items, Total: total}, nil
}

// GetTimeline returns a mixed activity timeline across all sources.
func (s *ActivityService) GetTimeline(ctx context.Context, page, limit int) (*Result, *errors.AppError) {
	rows, total, err := s.repo.FetchTimeline(page, limit)
	if err != nil {
		return nil, errors.ErrInternal("查询活动列表失败")
	}
	items := rowsToItems(rows)
	s.enrichGalgameItems(ctx, rows, items)
	s.hydrateActors(ctx, items)
	return &Result{Items: items, Total: total}, nil
}

// rowsToItems converts DB rows into response items (no enrichment yet).
// Identity is left blank — hydrated by hydrateActors after enrichGalgameItems
// has had a chance to inject GALGAME_CREATION actor IDs from the wiki brief.
func rowsToItems(rows []repository.ActivityRow) []dto.ActivityItem {
	items := make([]dto.ActivityItem, len(rows))
	for i, r := range rows {
		items[i] = dto.ActivityItem{
			UniqueID:  fmt.Sprintf("%s-%d", r.TypeStr, r.ID),
			Type:      r.TypeStr,
			Content:   r.Content,
			Link:      r.Link,
			Timestamp: r.Created,
			Actor: dto.Actor{
				ID: r.UserID,
			},
		}
	}
	return items
}

// enrichGalgameItems batch-fetches names for every galgame-scoped activity
// row from the wiki service and rewrites the content string per type:
//
//	GALGAME_CREATION          → "<game name>"
//	GALGAME_RESOURCE_CREATION → "在《<game name>》发布了下载资源"
//	GALGAME_RATING_CREATION   → "<game name> · <short summary>" (if summary)
//	GALGAME_COMMENT_CREATION  → "在《<game name>》<comment>"
//
// rows/items must be index-aligned; the caller guarantees this.
func (s *ActivityService) enrichGalgameItems(
	ctx context.Context,
	rows []repository.ActivityRow,
	items []dto.ActivityItem,
) {
	idSet := map[int]struct{}{}
	for _, r := range rows {
		if r.GalgameID > 0 {
			idSet[r.GalgameID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	briefMap, appErr := s.wikiGC.GetBatch(ctx, ids)
	if appErr != nil {
		return // graceful: leave raw content
	}

	pickName := func(id int) string {
		b, ok := briefMap[id]
		if !ok {
			return fmt.Sprintf("galgame#%d", id)
		}
		for _, n := range []string{b.NameZhCn, b.NameJaJp, b.NameEnUs, b.NameZhTw} {
			if n != "" {
				return n
			}
		}
		return fmt.Sprintf("galgame#%d", id)
	}

	for i, r := range rows {
		if r.GalgameID == 0 {
			continue
		}
		name := pickName(r.GalgameID)
		switch r.TypeStr {
		case "GALGAME_CREATION":
			items[i].Content = name
			// galgame table has no local user_id; pull the creator from
			// the wiki brief.
			if items[i].Actor.ID == 0 {
				if b, ok := briefMap[r.GalgameID]; ok {
					items[i].Actor.ID = b.UserID
				}
			}
		case "GALGAME_RESOURCE_CREATION":
			items[i].Content = fmt.Sprintf("在《%s》发布了下载资源", name)
		case "GALGAME_RATING_CREATION":
			if r.Content != "" {
				items[i].Content = fmt.Sprintf("%s · %s", name, r.Content)
			} else {
				items[i].Content = fmt.Sprintf("评价了《%s》", name)
			}
		case "GALGAME_COMMENT_CREATION":
			if r.Content != "" {
				items[i].Content = fmt.Sprintf("在《%s》%s", name, r.Content)
			} else {
				items[i].Content = fmt.Sprintf("评论了《%s》", name)
			}
		}
	}
}

// hydrateActors batch-fetches identity (name/avatar) from OAuth for every
// non-zero actor id and writes it back into the items. Runs after
// enrichGalgameItems so GALGAME_CREATION rows whose actor was injected from
// the wiki brief are also hydrated.
func (s *ActivityService) hydrateActors(ctx context.Context, items []dto.ActivityItem) {
	uids := userclient.CollectIDs(items, func(it dto.ActivityItem) int { return it.Actor.ID })
	if len(uids) == 0 {
		return
	}
	userMap := s.userClient.Hydrate(ctx, uids)
	for i := range items {
		u := userMap[items[i].Actor.ID]
		items[i].Actor.Name = u.Name
		items[i].Actor.Avatar = u.Avatar
	}
}
