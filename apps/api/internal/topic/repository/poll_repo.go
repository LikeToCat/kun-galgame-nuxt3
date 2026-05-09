package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type PollRepository struct {
	db *gorm.DB
}

func NewPollRepository(db *gorm.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) DB() *gorm.DB {
	return r.db
}

func (r *PollRepository) FindByID(id int) (*model.TopicPoll, error) {
	var poll model.TopicPoll
	err := r.db.First(&poll, id).Error
	return &poll, err
}

func (r *PollRepository) FindByTopicID(topicID int) ([]model.TopicPoll, error) {
	var polls []model.TopicPoll
	err := r.db.Where("topic_id = ?", topicID).Order("created DESC").Find(&polls).Error
	return polls, err
}

func (r *PollRepository) CountByTopicID(topicID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicPoll{}).Where("topic_id = ?", topicID).Count(&count).Error
	return count, err
}

func (r *PollRepository) FindOptionsByPollID(pollID int) ([]model.TopicPollOption, error) {
	var options []model.TopicPollOption
	err := r.db.Where("poll_id = ?", pollID).Order("id ASC").Find(&options).Error
	return options, err
}

func (r *PollRepository) FindUserVoteOptionIDs(pollID, userID int) ([]int, error) {
	var optionIDs []int
	err := r.db.Model(&model.TopicPollVote{}).
		Where("poll_id = ? AND user_id = ?", pollID, userID).
		Pluck("option_id", &optionIDs).Error
	return optionIDs, err
}

func (r *PollRepository) HasUserVoted(pollID, userID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicPollVote{}).
		Where("poll_id = ? AND user_id = ?", pollID, userID).
		Count(&count).Error
	return count > 0, err
}

// FindDistinctVoterIDs returns up to `limit` distinct user IDs that have voted
// on the poll. Identity is hydrated by the service layer via userclient.
func (r *PollRepository) FindDistinctVoterIDs(pollID, limit int) ([]int, error) {
	var ids []int
	err := r.db.Table("topic_poll_vote").
		Distinct("user_id").
		Where("poll_id = ?", pollID).
		Limit(limit).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *PollRepository) CountDistinctVoters(pollID int) (int, error) {
	var count int64
	err := r.db.Model(&model.TopicPollVote{}).
		Where("poll_id = ?", pollID).
		Distinct("user_id").
		Count(&count).Error
	return int(count), err
}

func (r *PollRepository) CountTotalVotes(pollID int) (int, error) {
	var count int64
	err := r.db.Model(&model.TopicPollVote{}).
		Where("poll_id = ?", pollID).
		Count(&count).Error
	return int(count), err
}

// ──────────────────────────────────────────
// Vote log
// ──────────────────────────────────────────

type VoteLogRow struct {
	ID         int
	UserID     int
	OptionText string
	CreatedAt  time.Time
}

func (r *PollRepository) FindVoteLogs(pollID, page, limit int) ([]VoteLogRow, int64, error) {
	var rows []VoteLogRow
	var total int64

	r.db.Model(&model.TopicPollVote{}).Where("poll_id = ?", pollID).Count(&total)

	err := r.db.Table("topic_poll_vote v").
		Select(`v.id, v.user_id,
			o.text AS option_text, v.created AS created_at`).
		Joins("JOIN topic_poll_option o ON o.id = v.option_id").
		Where("v.poll_id = ?", pollID).
		Order("v.created DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	return rows, total, err
}

// ──────────────────────────────────────────
// Tx-aware write operations (for the service coordinator)
// ──────────────────────────────────────────

// CreatePoll inserts a TopicPoll row inside the caller tx.
func (r *PollRepository) CreatePoll(tx *gorm.DB, poll *model.TopicPoll) error {
	return tx.Create(poll).Error
}

// CreatePollOption inserts a TopicPollOption row inside the caller tx.
func (r *PollRepository) CreatePollOption(tx *gorm.DB, opt *model.TopicPollOption) error {
	return tx.Create(opt).Error
}

// TouchTopicStatusUpdateTime bumps topic.status_update_time for poll activity.
func (r *PollRepository) TouchTopicStatusUpdateTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).
		Updates(map[string]any{"status_update_time": t}).Error
}

// DeleteUserVotes removes all of a user's votes for a given poll.
func (r *PollRepository) DeleteUserVotes(tx *gorm.DB, pollID, userID int) error {
	return tx.Where("poll_id = ? AND user_id = ?", pollID, userID).
		Delete(&model.TopicPollVote{}).Error
}

// AdjustOptionVoteCount adjusts topic_poll_option.vote_count by delta.
func (r *PollRepository) AdjustOptionVoteCount(tx *gorm.DB, optionID, delta int) error {
	return tx.Model(&model.TopicPollOption{}).Where("id = ?", optionID).
		Update("vote_count", gorm.Expr("vote_count + ?", delta)).Error
}

// CreateVote inserts a TopicPollVote row inside the caller tx.
func (r *PollRepository) CreateVote(tx *gorm.DB, pollID, optionID, userID int) error {
	return tx.Create(&model.TopicPollVote{
		PollID: pollID, OptionID: optionID, UserID: userID,
	}).Error
}

// DeletePollCascade removes a poll's votes, options, then the poll row itself.
func (r *PollRepository) DeletePollCascade(tx *gorm.DB, pollID int) error {
	if err := tx.Where("poll_id = ?", pollID).Delete(&model.TopicPollVote{}).Error; err != nil {
		return err
	}
	if err := tx.Where("poll_id = ?", pollID).Delete(&model.TopicPollOption{}).Error; err != nil {
		return err
	}
	return tx.Delete(&model.TopicPoll{}, pollID).Error
}

// UpdatePollFields patches poll scalar columns inside a tx.
func (r *PollRepository) UpdatePollFields(tx *gorm.DB, pollID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Model(&model.TopicPoll{}).Where("id = ?", pollID).Updates(fields).Error
}

// FindOptionsByIDs loads poll options by primary key.
func (r *PollRepository) FindOptionsByIDs(ids []int) ([]model.TopicPollOption, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var opts []model.TopicPollOption
	err := r.db.Where("id IN ?", ids).Find(&opts).Error
	return opts, err
}

// UpdateOptionText patches the text of a single option.
func (r *PollRepository) UpdateOptionText(tx *gorm.DB, optionID int, text string) error {
	return tx.Model(&model.TopicPollOption{}).Where("id = ?", optionID).
		Update("text", text).Error
}

// DeleteOptionsByIDs removes options by primary key (caller verifies no votes).
func (r *PollRepository) DeleteOptionsByIDs(tx *gorm.DB, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("id IN ?", ids).Delete(&model.TopicPollOption{}).Error
}
