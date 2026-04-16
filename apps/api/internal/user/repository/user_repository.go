package repository

import (
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

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

// ──────────────────────────────────────────
// Stats & counts
// ──────────────────────────────────────────

type UserStats = model.UserStats

func (r *UserRepository) GetUserStats(uid int) (*model.UserStats, error) {
	var stats model.UserStats

	r.db.Model(&model.User{}).
		Select("(SELECT COUNT(*) FROM topic WHERE user_id = ?) AS topic_count", uid).
		Scan(&stats)
	r.db.Model(&model.User{}).
		Select("(SELECT COUNT(*) FROM topic_reply WHERE user_id = ?) AS reply_count", uid).
		Scan(&stats)
	r.db.Model(&model.User{}).
		Select("(SELECT COUNT(*) FROM galgame WHERE user_id = ?) AS galgame_count", uid).
		Scan(&stats)
	r.db.Model(&model.User{}).
		Select("(SELECT COUNT(*) FROM topic_like WHERE user_id = ?) AS like_count", uid).
		Scan(&stats)

	return &stats, nil
}

func (r *UserRepository) CountUnreadMessages(uid int) (int64, error) {
	var count int64
	err := r.db.Table("message").
		Where("receiver_id = ? AND status = 'unread'", uid).
		Count(&count).Error
	return count, err
}

func (r *UserRepository) CountUnreadSystemMessages() (int64, error) {
	var count int64
	err := r.db.Table("system_message").
		Where("status = 'unread'").
		Count(&count).Error
	return count, err
}

func (r *UserRepository) CountUnreadChatMessages(uid int) (int64, error) {
	var count int64
	err := r.db.Table("chat_message").
		Where("sender_id != ?", uid).
		Where("chat_room_id IN (SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?)", uid).
		Where("id NOT IN (SELECT chat_message_id FROM chat_message_read_by WHERE user_id = ?)", uid).
		Count(&count).Error
	return count, err
}

// ──────────────────────────────────────────
// User galgames / topics queries
// ──────────────────────────────────────────

func (r *UserRepository) FindUserGalgameIDs(uid int, queryType string, page, limit int) ([]int, int64, error) {
	offset := (page - 1) * limit
	var total int64

	baseQuery := r.db.Table("galgame").Select("galgame.id")

	switch queryType {
	case "galgame_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_like ON galgame_like.galgame_id = galgame.id").
			Where("galgame_like.user_id = ?", uid)
	case "galgame_favorite":
		baseQuery = baseQuery.
			Joins("JOIN galgame_favorite ON galgame_favorite.galgame_id = galgame.id").
			Where("galgame_favorite.user_id = ?", uid)
	case "galgame_contribute":
		// contributor table was moved to wiki; query not supported locally
		return []int{}, 0, nil
	default:
		// "galgame" — user_id no longer on local table; not supported
		return []int{}, 0, nil
	}

	baseQuery.Count(&total)

	type idRow struct {
		ID int `gorm:"column:id"`
	}
	var rows []idRow
	err := baseQuery.Order("galgame.created DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, total, nil
}

func (r *UserRepository) FindUserTopics(uid int, queryType string, page, limit int) ([]dto.UserTopic, int64, error) {
	offset := (page - 1) * limit
	var results []dto.UserTopic
	var total int64

	baseQuery := r.db.Table("topic").
		Select("topic.id, topic.title, topic.created")

	switch queryType {
	case "topic":
		baseQuery = baseQuery.Where("topic.user_id = ?", uid)
	case "topic_like":
		baseQuery = baseQuery.
			Joins("JOIN topic_like ON topic_like.topic_id = topic.id").
			Where("topic_like.user_id = ?", uid)
	case "topic_upvote":
		baseQuery = baseQuery.
			Joins("JOIN topic_upvote ON topic_upvote.topic_id = topic.id").
			Where("topic_upvote.user_id = ?", uid)
	case "topic_favorite":
		baseQuery = baseQuery.
			Joins("JOIN topic_favorite ON topic_favorite.topic_id = topic.id").
			Where("topic_favorite.user_id = ?", uid)
	default:
		baseQuery = baseQuery.Where("topic.user_id = ?", uid)
	}

	baseQuery.Count(&total)
	err := baseQuery.Order("topic.created DESC").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}
