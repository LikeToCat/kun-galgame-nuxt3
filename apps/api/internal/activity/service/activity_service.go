package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"kun-galgame-api/internal/activity/dto"
	"kun-galgame-api/internal/activity/repository"
	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/errors"
)

type ActivityService struct {
	repo   *repository.ActivityRepository
	wikiGC *galgameClient.GalgameClient
}

func NewActivityService(
	repo *repository.ActivityRepository,
	gc *galgameClient.GalgameClient,
) *ActivityService {
	return &ActivityService{repo: repo, wikiGC: gc}
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
	s.enrichGalgameItems(ctx, items)
	return &Result{Items: items, Total: total}, nil
}

// GetTimeline returns a mixed activity timeline across all sources.
func (s *ActivityService) GetTimeline(ctx context.Context, page, limit int) (*Result, *errors.AppError) {
	rows, total, err := s.repo.FetchTimeline(page, limit)
	if err != nil {
		return nil, errors.ErrInternal("查询活动列表失败")
	}
	items := rowsToItems(rows)
	s.enrichGalgameItems(ctx, items)
	return &Result{Items: items, Total: total}, nil
}

// rowsToItems converts DB rows into response items (no enrichment yet).
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
				ID: r.UserID, Name: r.UserName, Avatar: r.Avatar,
			},
		}
	}
	return items
}

// enrichGalgameItems fills in galgame names for GALGAME_CREATION items
// by batch-fetching from the wiki service, then fills any missing actor
// names by resolving the wiki-returned user_ids against the local user table.
func (s *ActivityService) enrichGalgameItems(ctx context.Context, items []dto.ActivityItem) {
	ids := make([]int, 0)
	for _, it := range items {
		if it.Type == "GALGAME_CREATION" && strings.HasPrefix(it.Content, "galgame#") {
			if id, err := strconv.Atoi(it.Content[len("galgame#"):]); err == nil {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}

	briefMap, appErr := s.wikiGC.GetBatch(ctx, ids)
	if appErr != nil {
		return // graceful: leave placeholder content
	}

	for i := range items {
		if items[i].Type == "GALGAME_CREATION" && strings.HasPrefix(items[i].Content, "galgame#") {
			if id, err := strconv.Atoi(items[i].Content[len("galgame#"):]); err == nil {
				if b, ok := briefMap[id]; ok {
					// Pick best available name
					name := b.NameZhCn
					if name == "" {
						name = b.NameJaJp
					}
					if name == "" {
						name = b.NameEnUs
					}
					if name == "" {
						name = b.NameZhTw
					}
					items[i].Content = name
					// Fill actor from wiki user_id
					if items[i].Actor.ID == 0 {
						items[i].Actor.ID = b.UserID
					}
				}
			}
		}
	}

	// Batch resolve user names for galgame creation items that still lack one.
	userIDs := make([]int, 0)
	for _, it := range items {
		if it.Type == "GALGAME_CREATION" && it.Actor.Name == "" && it.Actor.ID > 0 {
			userIDs = append(userIDs, it.Actor.ID)
		}
	}
	if len(userIDs) == 0 {
		return
	}
	users := s.repo.FindUsersByIDs(userIDs)
	userMap := make(map[int]repository.UserInfoRow, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}
	for i := range items {
		if items[i].Type == "GALGAME_CREATION" && items[i].Actor.Name == "" {
			if u, ok := userMap[items[i].Actor.ID]; ok {
				items[i].Actor.Name = u.Name
				items[i].Actor.Avatar = u.Avatar
			}
		}
	}
}
