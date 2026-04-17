package repository

import "gorm.io/gorm"

// UserBriefRepository exposes lightweight user lookups used for list
// enrichment (id/name/avatar) and for the floating hover card.
type UserBriefRepository struct {
	db *gorm.DB
}

func NewUserBriefRepository(db *gorm.DB) *UserBriefRepository {
	return &UserBriefRepository{db: db}
}

func (r *UserBriefRepository) DB() *gorm.DB { return r.db }

// UserBriefRow is the (id, name, avatar) projection for list enrichment.
type UserBriefRow struct {
	ID     int    `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Avatar string `gorm:"column:avatar"`
}

// FindUsersByIDs batch-loads lightweight user info keyed by id.
func (r *UserBriefRepository) FindUsersByIDs(ids []int) map[int]UserBriefRow {
	if len(ids) == 0 {
		return map[int]UserBriefRow{}
	}
	var rows []UserBriefRow
	r.db.Table(`"user"`).Select("id, name, avatar").
		Where("id IN ?", ids).Scan(&rows)
	out := make(map[int]UserBriefRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

// ──────────────────────────────────────────
// Floating hover card user row
// ──────────────────────────────────────────

// FloatingUserRow is the trimmed user projection used for the hover card.
type FloatingUserRow struct {
	ID          int    `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Avatar      string `gorm:"column:avatar"`
	Moemoepoint int    `gorm:"column:moemoepoint"`
	Bio         string `gorm:"column:bio"`
	Status      int    `gorm:"column:status"`
}

// FindFloatingUser loads the lightweight user row for the hover card.
func (r *UserBriefRepository) FindFloatingUser(uid int) (*FloatingUserRow, error) {
	var user FloatingUserRow
	if err := r.db.Table(`"user"`).Where("id = ?", uid).Scan(&user).Error; err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}
