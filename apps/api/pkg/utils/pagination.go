package utils

import "gorm.io/gorm"

type Pagination struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// Paginate is a GORM scope for offset-based pagination.
func Paginate(page, limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * limit
		return db.Offset(offset).Limit(limit)
	}
}
