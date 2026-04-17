package repository

import (
	"kun-galgame-api/internal/admin/dto"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CountAll returns the total user count.
func (r *UserRepository) CountAll() int64 {
	var total int64
	r.db.Table(`"user"`).Count(&total)
	return total
}

// FindPaginated returns paginated admin-view user rows.
func (r *UserRepository) FindPaginated(page, limit int) []dto.AdminUserRow {
	var users []dto.AdminUserRow
	r.db.Table(`"user"`).
		Select("id, name, avatar, status, created").
		Order("created DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&users)
	return users
}

// SearchByName returns up to 50 users whose name matches a LIKE query.
func (r *UserRepository) SearchByName(keyword string) []dto.AdminUserRow {
	var users []dto.AdminUserRow
	r.db.Table(`"user"`).
		Select("id, name, avatar, status, created").
		Where("name ILIKE ?", "%"+keyword+"%").
		Limit(50).
		Find(&users)
	return users
}
