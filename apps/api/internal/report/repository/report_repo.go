package repository

import (
	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Create inserts a new report.
func (r *ReportRepository) Create(report *adminModel.Report) error {
	return r.db.Create(report).Error
}
