package repository

import (
	"kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

// UserRepository owns the core user row: lookups (by id/email/name/OAuth),
// CRUD, account linking, moemoepoint / check-in / username changes.
//
// Per-user aggregates (stats, counts), list queries (topics/replies/...) and
// lightweight brief lookups live in sibling repos in this package:
//   - UserStatsRepository   (stats_repo.go)
//   - UserContentRepository (content_repo.go)
//   - UserBriefRepository   (brief_repo.go)
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Basic CRUD
// ──────────────────────────────────────────

func (r *UserRepository) FindByID(id int) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *UserRepository) FindByName(name string) (*model.User, error) {
	var user model.User
	err := r.db.Where("name = ?", name).First(&user).Error
	return &user, err
}

func (r *UserRepository) FindByOAuthSub(sub string) (*model.User, error) {
	var account model.OAuthAccount
	err := r.db.Where("sub = ?", sub).Preload("User").First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account.User, nil
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id int) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepository) UpdateField(uid int, field string, value any) error {
	return r.db.Model(&model.User{}).Where("id = ?", uid).Update(field, value).Error
}

// ──────────────────────────────────────────
// OAuth
// ──────────────────────────────────────────

func (r *UserRepository) CreateOAuthAccount(account *model.OAuthAccount) error {
	return r.db.Create(account).Error
}

func (r *UserRepository) LinkOAuthAccount(userID int, sub string) error {
	return r.db.Create(&model.OAuthAccount{
		UserID:   userID,
		Provider: "kun-oauth",
		Sub:      sub,
	}).Error
}

// ──────────────────────────────────────────
// User operations
// ──────────────────────────────────────────

func (r *UserRepository) IncrementMoemoepoint(userID int, amount int) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", amount)).Error
}

func (r *UserRepository) CheckIn(uid int, points int) error {
	return r.db.Model(&model.User{}).Where("id = ?", uid).
		Updates(map[string]any{
			"daily_check_in": 1,
			"moemoepoint":    gorm.Expr("moemoepoint + ?", points),
		}).Error
}

func (r *UserRepository) UsernameExists(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).
		Where("LOWER(name) = LOWER(?)", username).
		Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) UpdateUsernameWithCost(uid int, username string, cost int) error {
	return r.db.Model(&model.User{}).Where("id = ?", uid).
		Updates(map[string]any{
			"name":        username,
			"moemoepoint": gorm.Expr("moemoepoint - ?", cost),
		}).Error
}
