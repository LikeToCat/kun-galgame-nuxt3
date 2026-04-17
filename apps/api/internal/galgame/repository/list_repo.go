package repository

import (
	"strings"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// GalgameListRepository owns the paginated list query with resource-filter
// support (type/language/platform + include/exclude provider sets).
type GalgameListRepository struct {
	db *gorm.DB
}

func NewGalgameListRepository(db *gorm.DB) *GalgameListRepository {
	return &GalgameListRepository{db: db}
}

func (r *GalgameListRepository) DB() *gorm.DB { return r.db }

// allProviders is the closed set of provider names used by the exclude-only filter.
var allProviders = []string{
	"baidu", "aliyun", "quark", "pan123", "tianyiyun",
	"caiyun", "xunlei", "uc", "lanzou", "other",
}

// ListIDs returns galgame IDs matching the given filter, paginated and sorted.
// If hasResourceFilter (returned by HasResourceFilter) is true, a JOIN against
// galgame_resource is used; otherwise a simple galgame-only scan.
func (r *GalgameListRepository) ListIDs(f model.GalgameListFilter) (ids []int, total int64) {
	sortCol := "g.updated"
	switch f.SortField {
	case "time":
		sortCol = "g.updated"
	case "created":
		sortCol = "g.created"
	case "view":
		sortCol = "g.view"
	}

	if !hasResourceFilter(f) {
		r.db.Table("galgame g").Select("COUNT(*)").Scan(&total)
		type idRow struct {
			ID int `gorm:"column:id"`
		}
		var rows []idRow
		r.db.Table("galgame g").
			Select("g.id").
			Order(sortCol + " " + f.SortOrder).
			Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
			Scan(&rows)
		ids = make([]int, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		return
	}

	// Join with galgame_resource and apply filters
	inner := r.db.Table("galgame g").
		Select("DISTINCT g.id").
		Joins("JOIN galgame_resource gr ON gr.galgame_id = g.id")

	if f.Type != "" && f.Type != "all" {
		inner = inner.Where("gr.type = ?", f.Type)
	}
	if f.Language != "" && f.Language != "all" {
		inner = inner.Where("gr.language = ?", f.Language)
	}
	if f.Platform != "" && f.Platform != "all" {
		inner = inner.Where("gr.platform = ?", f.Platform)
	}
	if len(f.IncludeProviders) > 0 {
		inner = inner.Where("gr.provider && ?", providerArrayLit(f.IncludeProviders))
	}
	if len(f.ExcludeOnlyProviders) > 0 {
		allowed := providersExcluding(f.ExcludeOnlyProviders)
		if len(allowed) > 0 {
			inner = inner.Where("gr.provider && ?", providerArrayLit(allowed))
		}
	}

	r.db.Table("(?) AS sub", inner).Select("COUNT(*)").Scan(&total)

	type idRow struct {
		ID int `gorm:"column:id"`
	}
	var rows []idRow
	r.db.Table("galgame g").
		Select("g.id").
		Joins("JOIN galgame_resource gr ON gr.galgame_id = g.id").
		Where("gr.galgame_id IN (?)", inner).
		Group("g.id, " + sortCol).
		Order(sortCol + " " + f.SortOrder).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)

	ids = make([]int, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return
}

func hasResourceFilter(f model.GalgameListFilter) bool {
	return (f.Type != "" && f.Type != "all") ||
		(f.Language != "" && f.Language != "all") ||
		(f.Platform != "" && f.Platform != "all") ||
		len(f.IncludeProviders) > 0 ||
		len(f.ExcludeOnlyProviders) > 0
}

func providerArrayLit(providers []string) string {
	return "{" + strings.Join(providers, ",") + "}"
}

func providersExcluding(excluded []string) []string {
	exSet := map[string]bool{}
	for _, e := range excluded {
		exSet[e] = true
	}
	out := make([]string, 0, len(allProviders))
	for _, p := range allProviders {
		if !exSet[p] {
			out = append(out, p)
		}
	}
	return out
}
