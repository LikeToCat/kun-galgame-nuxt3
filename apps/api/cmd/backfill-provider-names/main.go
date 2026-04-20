// Backfill galgame_resource.provider_name for every existing resource.
//
// For each resource row, joins galgame_resource_link to collect every URL,
// runs the same DetectProviderNamesFromURLs classifier used by the live API,
// and writes the deduped display names into the new jsonb column.
//
// Idempotent: re-running overwrites with the latest computation, so it can
// also be used to refresh names after the classifier table is extended.
//
// Usage:
//
//	go run ./cmd/backfill-provider-names              # do it
//	go run ./cmd/backfill-provider-names --dry-run    # report-only, no writes
//	go run ./cmd/backfill-provider-names --batch=500  # tune batch size
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"
	"kun-galgame-api/pkg/utils"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Compute names but do not update rows")
	batchSize := flag.Int("batch", 500, "Number of resources to process per batch")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	total := int64(0)
	if err := db.Table("galgame_resource").Count(&total).Error; err != nil {
		slog.Error("统计资源总数失败", "error", err)
		os.Exit(1)
	}
	slog.Info("开始回填", "total", total, "batch", *batchSize, "dry_run", *dryRun)

	processed := 0
	updated := 0
	skipped := 0

	type resRow struct {
		ID int
	}
	type linkRow struct {
		ResourceID int    `gorm:"column:galgame_resource_id"`
		URL        string `gorm:"column:url"`
	}

	lastID := 0
	for {
		var rows []resRow
		err := db.Table("galgame_resource").
			Select("id").
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(*batchSize).
			Scan(&rows).Error
		if err != nil {
			slog.Error("拉取资源批次失败", "error", err, "lastID", lastID)
			os.Exit(1)
		}
		if len(rows) == 0 {
			break
		}

		ids := make([]int, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
		}

		var links []linkRow
		if err := db.Table("galgame_resource_link").
			Select("galgame_resource_id, url").
			Where("galgame_resource_id IN ?", ids).
			Scan(&links).Error; err != nil {
			slog.Error("拉取资源链接失败", "error", err)
			os.Exit(1)
		}

		urlsByResource := make(map[int][]string, len(rows))
		for _, l := range links {
			urlsByResource[l.ResourceID] = append(urlsByResource[l.ResourceID], l.URL)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			for _, r := range rows {
				names := utils.DetectProviderNamesFromURLs(urlsByResource[r.ID])
				if names == nil {
					names = []string{}
				}
				encoded, jerr := json.Marshal(names)
				if jerr != nil {
					return fmt.Errorf("marshal names for id=%d: %w", r.ID, jerr)
				}

				if *dryRun {
					slog.Debug("dry-run", "id", r.ID, "names", names)
					continue
				}

				if err := tx.Exec(
					"UPDATE galgame_resource SET provider_name = ?::jsonb WHERE id = ?",
					string(encoded), r.ID,
				).Error; err != nil {
					return fmt.Errorf("update id=%d: %w", r.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("批次更新失败", "error", err)
			os.Exit(1)
		}

		for _, r := range rows {
			processed++
			if len(urlsByResource[r.ID]) == 0 {
				skipped++
			} else {
				updated++
			}
			lastID = r.ID
		}

		slog.Info("批次完成",
			"processed", processed, "updated", updated,
			"skipped_no_links", skipped, "lastID", lastID,
		)
	}

	if *dryRun {
		fmt.Printf("dry-run 完成: 共 %d 行, 其中 %d 行无 link\n", processed, skipped)
	} else {
		fmt.Printf("回填完成: 共 %d 行, 其中 %d 行无 link (provider_name 写入 [])\n", processed, skipped)
	}
}
