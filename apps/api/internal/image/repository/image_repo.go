package repository

import (
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type ImageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// GetDailyCount returns the user's current daily image upload count.
func (r *ImageRepository) GetDailyCount(uid int) (int, error) {
	var u userModel.User
	err := r.db.Select("daily_image_count").First(&u, uid).Error
	return u.DailyImageCount, err
}

// IncrementDailyCount atomically increments the user's daily image upload count.
func (r *ImageRepository) IncrementDailyCount(uid int) {
	r.db.Model(&userModel.User{}).Where("id = ?", uid).
		Update("daily_image_count", gorm.Expr("daily_image_count + 1"))
}
