package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	direction := flag.String("dir", "up", "Migration direction: up or down")
	step := flag.Int("step", 0, "Number of migrations to run (0 = all)")
	migrationsDir := flag.String("path", "migrations", "Path to migration files")
	exclude := flag.String("exclude", "005,006,007", "Comma-separated migration prefixes to exclude (e.g. '005,006'). Use empty string to include all. Default skips the post-OAuth-migration steps (005 wiki cleanup, 006 resource provider, 007 user identity drop) — these must run explicitly via --only after the OAuth-side migrate-users + galgame service migrations have completed.")
	only := flag.String("only", "", "Comma-separated migration prefixes to run exclusively (e.g. '005')")
	flag.Parse()

	excludeSet := map[string]bool{}
	if *only == "" && *exclude != "" {
		for _, prefix := range strings.Split(*exclude, ",") {
			prefix = strings.TrimSpace(prefix)
			if prefix != "" {
				excludeSet[prefix] = true
			}
		}
	}
	onlySet := map[string]bool{}
	if *only != "" {
		for _, prefix := range strings.Split(*only, ",") {
			prefix = strings.TrimSpace(prefix)
			if prefix != "" {
				onlySet[prefix] = true
			}
		}
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("获取数据库连接失败", "error", err)
		os.Exit(1)
	}

	// Create migration tracking table
	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		slog.Error("创建迁移跟踪表失败", "error", err)
		os.Exit(1)
	}

	// Get applied migrations
	rows, err := sqlDB.Query("SELECT name FROM _migrations ORDER BY id")
	if err != nil {
		slog.Error("查询已应用迁移失败", "error", err)
		os.Exit(1)
	}
	defer rows.Close()

	applied := map[string]bool{}
	var appliedList []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		applied[name] = true
		appliedList = append(appliedList, name)
	}

	suffix := "." + *direction + ".sql"

	// Find migration files
	files, err := filepath.Glob(filepath.Join(*migrationsDir, "*"+suffix))
	if err != nil {
		slog.Error("查找迁移文件失败", "error", err)
		os.Exit(1)
	}
	sort.Strings(files)

	if *direction == "down" {
		// Reverse order for down migrations
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	ran := 0
	for _, file := range files {
		base := filepath.Base(file)
		name := strings.TrimSuffix(base, suffix)

		// Extract numeric prefix (e.g. "005" from "005_cleanup_wiki_managed_data")
		prefix := strings.SplitN(name, "_", 2)[0]

		// Filter by --only or --exclude
		if len(onlySet) > 0 {
			if !onlySet[prefix] {
				continue
			}
		} else if excludeSet[prefix] {
			slog.Info("跳过迁移 (excluded)", "file", base)
			continue
		}

		if *direction == "up" {
			if applied[name] {
				continue
			}
		} else {
			if !applied[name] {
				continue
			}
		}

		if *step > 0 && ran >= *step {
			break
		}

		slog.Info("执行迁移", "file", base, "direction", *direction)

		content, err := os.ReadFile(file)
		if err != nil {
			slog.Error("读取迁移文件失败", "file", base, "error", err)
			os.Exit(1)
		}

		_, err = sqlDB.Exec(string(content))
		if err != nil {
			slog.Error("执行迁移失败", "file", base, "error", err)
			os.Exit(1)
		}

		if *direction == "up" {
			_, err = sqlDB.Exec("INSERT INTO _migrations (name) VALUES ($1)", name)
		} else {
			_, err = sqlDB.Exec("DELETE FROM _migrations WHERE name = $1", name)
		}
		if err != nil {
			slog.Error("更新迁移记录失败", "file", base, "error", err)
			os.Exit(1)
		}

		ran++
		slog.Info("迁移完成", "file", base)
	}

	if ran == 0 {
		fmt.Println("没有待执行的迁移")
	} else {
		fmt.Printf("成功执行 %d 个迁移\n", ran)
	}
}
