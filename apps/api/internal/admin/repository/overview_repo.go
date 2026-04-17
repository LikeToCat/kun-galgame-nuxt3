package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type OverviewRepository struct {
	db *gorm.DB
}

func NewOverviewRepository(db *gorm.DB) *OverviewRepository {
	return &OverviewRepository{db: db}
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

// DailyStat is a single (date, count) row.
type DailyStat struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// CountTable returns the row count of a given table. The table name is
// produced internally by the service and is safe to interpolate.
func (r *OverviewRepository) CountTable(table string) int64 {
	var count int64
	r.db.Table(table).Count(&count)
	return count
}

// DailyCountsSince returns per-day counts for a given table since the given
// `since` timestamp. The table name is produced internally by the service and
// is safe to interpolate.
func (r *OverviewRepository) DailyCountsSince(table string, since any) []DailyStat {
	var stats []DailyStat
	r.db.Raw(fmt.Sprintf(`
		SELECT date_trunc('day', created)::date::text AS date, COUNT(*) AS count
		FROM %s WHERE created >= ? GROUP BY 1 ORDER BY 1
	`, table), since).Scan(&stats)
	return stats
}
