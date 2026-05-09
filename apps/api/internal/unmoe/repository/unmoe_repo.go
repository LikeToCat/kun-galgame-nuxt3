package repository

import (
	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

// UnmoeRepository owns the read query for the unmoe character translator.
type UnmoeRepository struct {
	db *gorm.DB
}

func NewUnmoeRepository(db *gorm.DB) *UnmoeRepository {
	return &UnmoeRepository{db: db}
}

// UnmoeRow is a log entry row. Identity (UserName/UserAvatar) is hydrated by
// the handler/service via userclient.
type UnmoeRow struct {
	ID       int    `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	Result   string `gorm:"column:result"`
	DescEnUs string `gorm:"column:desc_en_us"`
	DescJaJp string `gorm:"column:desc_ja_jp"`
	DescZhCn string `gorm:"column:desc_zh_cn"`
	DescZhTw string `gorm:"column:desc_zh_tw"`
	UserID   int    `gorm:"column:user_id"`
	Created  string `gorm:"column:created"`
}

// FindPaginated returns log rows ordered by created DESC plus a total.
func (r *UnmoeRepository) FindPaginated(page, limit int) ([]UnmoeRow, int64) {
	var rows []UnmoeRow
	var total int64

	r.db.Model(&adminModel.Unmoe{}).Count(&total)

	r.db.Table("unmoe u").
		Select(`u.id, u.name, u.result,
			u.desc_en_us, u.desc_ja_jp, u.desc_zh_cn, u.desc_zh_tw,
			u.user_id, u.created`).
		Order("u.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(&rows)

	return rows, total
}
