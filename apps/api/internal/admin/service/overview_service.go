package service

import (
	"context"
	"sort"
	"time"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
	galgameClient "kun-galgame-api/internal/galgame/client"
)

// OverviewService produces admin overview/stats responses by combining local
// DB counts with remote wiki-service stats.
type OverviewService struct {
	overviewRepo *repository.OverviewRepository
	wikiGC       *galgameClient.GalgameClient
}

func NewOverviewService(
	overviewRepo *repository.OverviewRepository,
	wikiGC *galgameClient.GalgameClient,
) *OverviewService {
	return &OverviewService{overviewRepo: overviewRepo, wikiGC: wikiGC}
}

// ──────────────────────────────────────────
// Table catalog (DB side)
// ──────────────────────────────────────────

type localModel struct {
	Name, Table, Label string
}

func localModels() []localModel {
	return []localModel{
		{"user", `"user"`, "用户"},
		{"topic", "topic", "话题"},
		{"topic_reply", "topic_reply", "话题回复"},
		{"topic_comment", "topic_comment", "话题评论"},
		{"galgame", "galgame", "Galgame"},
		{"galgame_resource", "galgame_resource", "Galgame 资源"},
		{"galgame_comment", "galgame_comment", "Galgame 评论"},
		{"galgame_website", "galgame_website", "Galgame 网站"},
		{"galgame_website_comment", "galgame_website_comment", "Galgame 网站评论"},
		{"chat_message", "chat_message", "聊天消息"},
	}
}

type wikiModel struct {
	Key, Label string
}

func wikiModels() []wikiModel {
	return []wikiModel{
		{"galgame_tag", "Galgame 标签"},
		{"galgame_official", "Galgame 会社"},
		{"galgame_engine", "Galgame 引擎"},
		{"galgame_series", "Galgame 系列"},
		{"galgame_link", "Galgame 链接"},
		{"galgame_pr", "Galgame PR"},
		{"galgame_revision", "Galgame 编辑历史"},
	}
}

// ──────────────────────────────────────────
// GetOverview — GET /admin/overview/all
// ──────────────────────────────────────────

func (s *OverviewService) GetOverview(ctx context.Context) []dto.OverviewItem {
	locals := localModels()
	wikis := wikiModels()

	items := make([]dto.OverviewItem, 0, len(locals)+len(wikis))
	for _, m := range locals {
		items = append(items, dto.OverviewItem{
			Name:  m.Name,
			Label: m.Label,
			Count: s.overviewRepo.CountTable(m.Table),
		})
	}

	// Merge wiki totals (non-blocking — on error we still emit zero rows).
	var totals map[string]int64
	if wikiStats, err := s.wikiGC.GetAdminStats(ctx, 1); err == nil && wikiStats != nil {
		totals = wikiStats.Totals
	}
	for _, m := range wikis {
		items = append(items, dto.OverviewItem{
			Name:  m.Key,
			Label: m.Label,
			Count: totals[m.Key],
		})
	}

	return items
}

// ──────────────────────────────────────────
// GetStats — GET /admin/overview/stats
// ──────────────────────────────────────────

func (s *OverviewService) GetStats(ctx context.Context, days int) []dto.DailyStatRow {
	if days == 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	locals := localModels()
	wikis := wikiModels()

	// date -> key -> count
	dateMap := make(map[string]map[string]int64)

	for _, t := range locals {
		stats := s.overviewRepo.DailyCountsSince(t.Table, since)
		for _, row := range stats {
			if dateMap[row.Date] == nil {
				dateMap[row.Date] = make(map[string]int64)
			}
			dateMap[row.Date][t.Name] = row.Count
		}
	}

	// Merge wiki daily stats (non-blocking).
	if wikiStats, err := s.wikiGC.GetAdminStats(ctx, days); err == nil && wikiStats != nil {
		for _, day := range wikiStats.Daily {
			date, _ := day["date"].(string)
			if date == "" {
				continue
			}
			if dateMap[date] == nil {
				dateMap[date] = make(map[string]int64)
			}
			for _, w := range wikis {
				v, ok := day[w.Key]
				if !ok {
					continue
				}
				switch n := v.(type) {
				case float64:
					dateMap[date][w.Key] = int64(n)
				case int64:
					dateMap[date][w.Key] = n
				}
			}
		}
	}

	// Build sorted flat array: [{date, user, topic, …}, …]
	allKeys := make([]string, 0, len(locals)+len(wikis))
	for _, t := range locals {
		allKeys = append(allKeys, t.Name)
	}
	for _, w := range wikis {
		allKeys = append(allKeys, w.Key)
	}

	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	result := make([]dto.DailyStatRow, len(dates))
	for i, d := range dates {
		row := dto.DailyStatRow{"date": d}
		for _, key := range allKeys {
			row[key] = dateMap[d][key]
		}
		result[i] = row
	}

	return result
}
