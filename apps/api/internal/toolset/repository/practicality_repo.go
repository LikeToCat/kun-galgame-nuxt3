package repository

import (
	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type PracticalityRepository struct {
	db *gorm.DB
}

func NewPracticalityRepository(db *gorm.DB) *PracticalityRepository {
	return &PracticalityRepository{db: db}
}

func (r *PracticalityRepository) DB() *gorm.DB { return r.db }

// RateCount holds a row from `SELECT rate, COUNT(*) GROUP BY rate`.
type RateCount struct {
	Rate  int   `json:"rate"`
	Count int64 `json:"count"`
}

// CountsByToolset returns the per-rate distribution for a toolset.
// The returned slice only contains rates with count > 0.
func (r *PracticalityRepository) CountsByToolset(toolsetID int) []RateCount {
	var counts []RateCount
	r.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", toolsetID).
		Select("rate, COUNT(*) as count").
		Group("rate").
		Scan(&counts)
	return counts
}

// AverageRate returns the average rating (0 if none exist).
func (r *PracticalityRepository) AverageRate(toolsetID int) float64 {
	var avg float64
	r.db.Model(&model.GalgameToolsetPracticality{}).
		Where("toolset_id = ?", toolsetID).
		Select("COALESCE(AVG(rate), 0)").Scan(&avg)
	return avg
}

// AveragesForToolsets returns a map[toolsetID]avg for a batch of toolsets.
func (r *PracticalityRepository) AveragesForToolsets(toolsetIDs []int) map[int]float64 {
	if len(toolsetIDs) == 0 {
		return map[int]float64{}
	}
	type row struct {
		ToolsetID int
		Avg       float64
	}
	var rows []row
	r.db.Model(&model.GalgameToolsetPracticality{}).
		Select("toolset_id, COALESCE(AVG(rate), 0) AS avg").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows)
	out := make(map[int]float64, len(rows))
	for _, r := range rows {
		out[r.ToolsetID] = r.Avg
	}
	return out
}

// FindUserRating returns the user's rating for a toolset, if any.
// Returns (nil, nil) when the user has not rated.
func (r *PracticalityRepository) FindUserRating(toolsetID, userID int) (*model.GalgameToolsetPracticality, error) {
	var p model.GalgameToolsetPracticality
	err := r.db.Where("toolset_id = ? AND user_id = ?", toolsetID, userID).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Upsert creates or updates the user's rating for a toolset.
func (r *PracticalityRepository) Upsert(toolsetID, userID, rate int) error {
	existing, err := r.FindUserRating(toolsetID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return r.db.Create(&model.GalgameToolsetPracticality{
			Rate:      rate,
			UserID:    userID,
			ToolsetID: toolsetID,
		}).Error
	}
	return r.db.Model(existing).Update("rate", rate).Error
}
