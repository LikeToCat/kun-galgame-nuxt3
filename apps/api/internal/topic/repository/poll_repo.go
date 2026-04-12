package repository

import (
	"kun-galgame-api/internal/topic/dto"
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

func (r *PollRepository) FindDistinctVoters(pollID, limit int) ([]dto.KunUser, error) {
	var voters []dto.KunUser
	err := r.db.Table("topic_poll_vote").
		Select(`DISTINCT ON ("user".id) "user".id, "user".name, "user".avatar`).
		Joins(`JOIN "user" ON "user".id = topic_poll_vote.user_id`).
		Where("topic_poll_vote.poll_id = ?", pollID).
		Limit(limit).
		Find(&voters).Error
	return voters, err
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
	UserName   string
	UserAvatar string
	OptionText string
	CreatedAt  string
}

func (r *PollRepository) FindVoteLogs(pollID, page, limit int) ([]VoteLogRow, int64, error) {
	var rows []VoteLogRow
	var total int64

	r.db.Model(&model.TopicPollVote{}).Where("poll_id = ?", pollID).Count(&total)

	err := r.db.Table("topic_poll_vote v").
		Select(`v.id, v.user_id, "user".name AS user_name, "user".avatar AS user_avatar,
			o.text AS option_text, v.created AS created_at`).
		Joins(`JOIN "user" ON "user".id = v.user_id`).
		Joins("JOIN topic_poll_option o ON o.id = v.option_id").
		Where("v.poll_id = ?", pollID).
		Order("v.created DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	return rows, total, err
}
