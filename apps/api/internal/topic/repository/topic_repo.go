package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

// TopicRepository owns the core topic lifecycle: single-row CRUD, daily-limit
// count, interaction-check predicates, interaction row toggles + counter
// adjustments, and the topic-author projection.
//
// Sibling repos in this package own the list queries and the tag/section
// relations:
//   - TopicListRepository     (list_repo.go)
//   - TopicTaxonomyRepository (taxonomy_repo.go)
type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (r *TopicRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Core CRUD
// ──────────────────────────────────────────

func (r *TopicRepository) FindByID(id int) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.First(&topic, id).Error
	return &topic, err
}

func (r *TopicRepository) Create(topic *model.Topic) error {
	return r.db.Create(topic).Error
}

func (r *TopicRepository) UpdateFields(id int, fields map[string]any) error {
	return r.db.Model(&model.Topic{}).Where("id = ?", id).Updates(fields).Error
}

func (r *TopicRepository) IncrementView(id int) error {
	return r.db.Model(&model.Topic{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1")).Error
}

// ──────────────────────────────────────────
// Interaction checks
// ──────────────────────────────────────────

func (r *TopicRepository) HasUserLiked(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicLike{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserDisliked(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicDislike{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserFavorited(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicFavorite{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserUpvoted(uid, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicUpvote{}).Where("user_id = ? AND topic_id = ?", uid, topicID).Count(&count).Error
	return count > 0, err
}

// ──────────────────────────────────────────
// Daily limit
// ──────────────────────────────────────────

func (r *TopicRepository) CountTodayTopicsByUser(tx *gorm.DB, uid int) (int64, error) {
	var count int64
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := tx.Model(&model.Topic{}).
		Where("user_id = ? AND created >= ?", uid, oneDayAgo).
		Count(&count).Error
	return count, err
}

// ──────────────────────────────────────────
// Poll existence check
// ──────────────────────────────────────────

func (r *TopicRepository) HasPoll(topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicPoll{}).Where("topic_id = ?", topicID).Count(&count).Error
	return count > 0, err
}

// ──────────────────────────────────────────
// Tx-aware write operations (for the service coordinator)
// ──────────────────────────────────────────

// FindByIDTx loads a topic inside a transaction.
func (r *TopicRepository) FindByIDTx(tx *gorm.DB, topicID int) (*model.Topic, error) {
	var topic model.Topic
	err := tx.First(&topic, topicID).Error
	return &topic, err
}

// CreateTopic inserts a Topic row inside the caller tx.
func (r *TopicRepository) CreateTopic(tx *gorm.DB, topic *model.Topic) error {
	return tx.Create(topic).Error
}

// UpdateTopicFields updates arbitrary Topic columns inside the caller tx.
func (r *TopicRepository) UpdateTopicFields(tx *gorm.DB, topicID int, fields map[string]any) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).Updates(fields).Error
}

// TouchStatusUpdateTime bumps status_update_time for a topic.
func (r *TopicRepository) TouchStatusUpdateTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).
		Updates(map[string]any{"status_update_time": t}).Error
}

// ──────────────────────────────────────────
// Interaction rows (tx-aware)
// ──────────────────────────────────────────

// FindTopicLike returns an existing TopicLike row (or gorm.ErrRecordNotFound).
func (r *TopicRepository) FindTopicLike(tx *gorm.DB, uid, topicID int) (*model.TopicLike, error) {
	var existing model.TopicLike
	err := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing).Error
	return &existing, err
}

// CreateTopicLike inserts a TopicLike row.
func (r *TopicRepository) CreateTopicLike(tx *gorm.DB, uid, topicID int) error {
	return tx.Create(&model.TopicLike{UserID: uid, TopicID: topicID}).Error
}

// DeleteTopicLike removes a previously fetched TopicLike row.
func (r *TopicRepository) DeleteTopicLike(tx *gorm.DB, like *model.TopicLike) error {
	return tx.Delete(like).Error
}

// FindTopicDislike returns an existing TopicDislike row.
func (r *TopicRepository) FindTopicDislike(tx *gorm.DB, uid, topicID int) (*model.TopicDislike, error) {
	var existing model.TopicDislike
	err := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing).Error
	return &existing, err
}

// CreateTopicDislike inserts a TopicDislike row.
func (r *TopicRepository) CreateTopicDislike(tx *gorm.DB, uid, topicID int) error {
	return tx.Create(&model.TopicDislike{UserID: uid, TopicID: topicID}).Error
}

// DeleteTopicDislike removes a previously fetched TopicDislike row.
func (r *TopicRepository) DeleteTopicDislike(tx *gorm.DB, dislike *model.TopicDislike) error {
	return tx.Delete(dislike).Error
}

// FindTopicFavorite returns an existing TopicFavorite row.
func (r *TopicRepository) FindTopicFavorite(tx *gorm.DB, uid, topicID int) (*model.TopicFavorite, error) {
	var existing model.TopicFavorite
	err := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing).Error
	return &existing, err
}

// CreateTopicFavorite inserts a TopicFavorite row.
func (r *TopicRepository) CreateTopicFavorite(tx *gorm.DB, uid, topicID int) error {
	return tx.Create(&model.TopicFavorite{UserID: uid, TopicID: topicID}).Error
}

// DeleteTopicFavorite removes a previously fetched TopicFavorite row.
func (r *TopicRepository) DeleteTopicFavorite(tx *gorm.DB, fav *model.TopicFavorite) error {
	return tx.Delete(fav).Error
}

// CreateTopicUpvote inserts a TopicUpvote row (duplicates allowed).
func (r *TopicRepository) CreateTopicUpvote(tx *gorm.DB, uid, topicID int) error {
	return tx.Create(&model.TopicUpvote{UserID: uid, TopicID: topicID}).Error
}

// AdjustLikeCount adjusts topic.like_count by delta.
func (r *TopicRepository) AdjustLikeCount(tx *gorm.DB, topicID, delta int) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// AdjustDislikeCount adjusts topic.dislike_count by delta.
func (r *TopicRepository) AdjustDislikeCount(tx *gorm.DB, topicID, delta int) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).
		Update("dislike_count", gorm.Expr("dislike_count + ?", delta)).Error
}

// AdjustFavoriteCount adjusts topic.favorite_count by delta.
func (r *TopicRepository) AdjustFavoriteCount(tx *gorm.DB, topicID, delta int) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).
		Update("favorite_count", gorm.Expr("favorite_count + ?", delta)).Error
}

// ApplyUpvoteCountAndTime bumps upvote_count and sets upvote_time / status_update_time.
func (r *TopicRepository) ApplyUpvoteCountAndTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).Updates(map[string]any{
		"upvote_count":       gorm.Expr("upvote_count + 1"),
		"status_update_time": t,
		"upvote_time":        &t,
	}).Error
}

// ──────────────────────────────────────────
// User lookup (avoids cross-module dependency for small projections)
// ──────────────────────────────────────────

// TopicAuthorUser is the minimal user projection needed on the topic detail page.
type TopicAuthorUser struct {
	ID          int    `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Avatar      string `gorm:"column:avatar"`
	Moemoepoint int    `gorm:"column:moemoepoint"`
}

// FindTopicAuthor loads minimal user fields for the topic author card.
func (r *TopicRepository) FindTopicAuthor(userID int) (*TopicAuthorUser, error) {
	var u TopicAuthorUser
	err := r.db.Table(`"user"`).Select("id, name, avatar, moemoepoint").
		Where("id = ?", userID).Scan(&u).Error
	return &u, err
}
