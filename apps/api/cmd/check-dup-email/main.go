package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type dupEmail struct {
	Email string
	Count int64
}

type userRow struct {
	ID    int
	Name  string
	Email string
}

var keepIDs = map[int]bool{
	18640: true, // Yamasune
	17592: true, // momoda
}

func main() {
	_ = godotenv.Load()
	dsn := os.Getenv("KUN_DATABASE_URL")
	if dsn == "" {
		fmt.Println("KUN_DATABASE_URL 未设置")
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	var dups []dupEmail
	db.Raw(`
		SELECT LOWER(email) AS email, COUNT(*) AS count
		FROM "user"
		GROUP BY LOWER(email)
		HAVING COUNT(*) > 1
		ORDER BY count DESC
	`).Scan(&dups)

	if len(dups) == 0 {
		fmt.Println("没有重复邮箱")
		return
	}

	fmt.Printf("发现 %d 组重复邮箱，开始处理...\n\n", len(dups))

	totalFixed := 0

	for _, d := range dups {
		var users []userRow
		db.Raw(`
			SELECT id, name, email FROM "user"
			WHERE LOWER(email) = ? ORDER BY id
		`, d.Email).Scan(&users)

		keepID := 0
		for _, u := range users {
			if keepIDs[u.ID] {
				keepID = u.ID
				break
			}
		}
		if keepID == 0 {
			keepID = users[0].ID
		}

		for _, u := range users {
			if u.ID == keepID {
				fmt.Printf("  保留  ID=%-6d  Name=%-20s  Email=%s\n", u.ID, u.Name, u.Email)
				continue
			}

			domain := ""
			if atIdx := strings.LastIndex(u.Email, "@"); atIdx >= 0 {
				domain = strings.ToLower(u.Email[atIdx:])
			} else {
				domain = "@dedup.local"
			}
			newEmail := randomPrefix(20) + domain

			err := db.Exec(`UPDATE "user" SET email = ? WHERE id = ?`, newEmail, u.ID).Error
			if err != nil {
				fmt.Printf("  失败  ID=%-6d  Error: %v\n", u.ID, err)
				continue
			}

			fmt.Printf("  改写  ID=%-6d  Name=%-20s  %s → %s\n", u.ID, u.Name, u.Email, newEmail)
			totalFixed++
		}
		fmt.Println()
	}

	fmt.Printf("完成，共修改 %d 个用户的邮箱\n", totalFixed)
}

func randomPrefix(n int) string {
	b := make([]byte, (n+1)/2)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
