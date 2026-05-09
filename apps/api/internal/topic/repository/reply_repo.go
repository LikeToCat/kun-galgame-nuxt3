package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

// ReplyRepository owns topic-reply rows: CRUD, paginated lists, target links,
// reply-level interactions (like/dislike + count adjustments), cascade delete
// and the reply-author brief projection.
//
// Comment rows (TopicComment) have their own sibling repo in this package:
//   - CommentRepository (comment_repo.go)
type ReplyRepository struct {
	db *gorm.DB
}

func NewReplyRepository(db *gorm.DB) *ReplyRepository {
	return &ReplyRepository{db: db}
}

func (r *ReplyRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Core CRUD
// ──────────────────────────────────────────

func (r *ReplyRepository) FindByID(id int) (*model.TopicReply, error) {
	var reply model.TopicReply
	err := r.db.First(&reply, id).Error
	return &reply, err
}

func (r *ReplyRepository) GetMaxFloor(tx *gorm.DB, topicID int) (int, error) {
	var maxFloor *int
	err := tx.Model(&model.TopicReply{}).
		Where("topic_id = ?", topicID).
		Select("COALESCE(MAX(floor), 0)").
		Scan(&maxFloor).Error
	if err != nil || maxFloor == nil {
		return 0, err
	}
	return *maxFloor, nil
}

type ReplyRow struct {
	model.TopicReply
	UserName        string
	UserAvatar      string
	UserMoemoepoint int
}

func (r *ReplyRepository) FindRepliesPaginated(
	topicID int,
	excludeIDs []int,
	page, limit int,
	sortOrder string,
) ([]ReplyRow, error) {
	var rows []ReplyRow
	query := r.db.Table("topic_reply").
		Select(`topic_reply.*`).
		Where("topic_reply.topic_id = ?", topicID)

	if len(excludeIDs) > 0 {
		query = query.Where("topic_reply.id NOT IN ?", excludeIDs)
	}

	err := query.
		Order("topic_reply.floor " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *ReplyRepository) FindRepliesByIDs(ids []int) ([]ReplyRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []ReplyRow
	err := r.db.Table("topic_reply").
		Select(`topic_reply.*`).
		Where("topic_reply.id IN ?", ids).
		Find(&rows).Error
	return rows, err
}

// ──────────────────────────────────────────
// Targets
// ──────────────────────────────────────────

type TargetRow struct {
	model.TopicReplyTarget
	TargetFloor      int
	TargetContent    string
	TargetUserID     int
	TargetUserName   string
	TargetUserAvatar string
}

func (r *ReplyRepository) FindTargetsByReplyIDs(replyIDs []int) (map[int][]TargetRow, error) {
	if len(replyIDs) == 0 {
		return make(map[int][]TargetRow), nil
	}
	var rows []TargetRow
	err := r.db.Table("topic_reply_target").
		Select(`topic_reply_target.*,
			tr.floor AS target_floor,
			tr.content AS target_content,
			tr.user_id AS target_user_id`).
		Joins("LEFT JOIN topic_reply tr ON tr.id = topic_reply_target.target_reply_id").
		Where("topic_reply_target.reply_id IN ?", replyIDs).
		Order("tr.floor ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]TargetRow)
	for _, row := range rows {
		result[row.ReplyID] = append(result[row.ReplyID], row)
	}
	return result, nil
}

// ──────────────────────────────────────────
// Interaction status (batch)
// ──────────────────────────────────────────

func (r *ReplyRepository) FindReplyLikeStatus(uid int, replyIDs []int) (map[int]bool, error) {
	return findInteractionStatus(r.db, "topic_reply_like", "topic_reply_id", uid, replyIDs)
}

func (r *ReplyRepository) FindReplyDislikeStatus(uid int, replyIDs []int) (map[int]bool, error) {
	return findInteractionStatus(r.db, "topic_reply_dislike", "topic_reply_id", uid, replyIDs)
}

// findInteractionStatus is shared by both ReplyRepository and CommentRepository
// (see comment_repo.go) to resolve a user's boolean interaction state across
// an ID batch.
func findInteractionStatus(db *gorm.DB, table, fkCol string, uid int, ids []int) (map[int]bool, error) {
	if len(ids) == 0 || uid == 0 {
		return make(map[int]bool), nil
	}
	var foundIDs []int
	err := db.Table(table).
		Where("user_id = ? AND "+fkCol+" IN ?", uid, ids).
		Pluck(fkCol, &foundIDs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(foundIDs))
	for _, id := range foundIDs {
		result[id] = true
	}
	return result, nil
}

// ──────────────────────────────────────────
// Cascade delete helpers
// ──────────────────────────────────────────

// CollectCascadeReplyIDs finds all replies that transitively target the given root IDs.
func (r *ReplyRepository) CollectCascadeReplyIDs(tx *gorm.DB, rootIDs []int) ([]int, error) {
	allIDs := make(map[int]bool)
	for _, id := range rootIDs {
		allIDs[id] = true
	}

	queue := rootIDs
	for len(queue) > 0 {
		var childIDs []int
		err := tx.Table("topic_reply_target").
			Where("target_reply_id IN ?", queue).
			Distinct("reply_id").
			Pluck("reply_id", &childIDs).Error
		if err != nil {
			return nil, err
		}

		queue = nil
		for _, cid := range childIDs {
			if !allIDs[cid] {
				allIDs[cid] = true
				queue = append(queue, cid)
			}
		}
	}

	result := make([]int, 0, len(allIDs))
	for id := range allIDs {
		result = append(result, id)
	}
	return result, nil
}

func (r *ReplyRepository) DeleteRepliesByIDs(tx *gorm.DB, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	// Delete targets, comments, likes, dislikes first
	tx.Where("reply_id IN ?", ids).Delete(&model.TopicReplyTarget{})
	tx.Where("target_reply_id IN ?", ids).Delete(&model.TopicReplyTarget{})

	// Delete comment likes for comments on these replies
	tx.Exec("DELETE FROM topic_comment_like WHERE topic_comment_id IN (SELECT id FROM topic_comment WHERE topic_reply_id IN ?)", ids)
	tx.Where("topic_reply_id IN ?", ids).Delete(&model.TopicComment{})
	tx.Where("topic_reply_id IN ?", ids).Delete(&model.TopicReplyLike{})
	tx.Where("topic_reply_id IN ?", ids).Delete(&model.TopicReplyDislike{})

	return tx.Where("id IN ?", ids).Delete(&model.TopicReply{}).Error
}

// CountReplyRelated returns counts used for moemoepoint penalty calculation.
func (r *ReplyRepository) CountReplyRelated(replyID int) (commentCount, likeCount, targetCount, targetByCount int64, err error) {
	r.db.Model(&model.TopicComment{}).Where("topic_reply_id = ?", replyID).Count(&commentCount)
	r.db.Model(&model.TopicReplyLike{}).Where("topic_reply_id = ?", replyID).Count(&likeCount)
	r.db.Model(&model.TopicReplyTarget{}).Where("reply_id = ?", replyID).Count(&targetCount)
	r.db.Model(&model.TopicReplyTarget{}).Where("target_reply_id = ?", replyID).Count(&targetByCount)
	return
}

// ──────────────────────────────────────────
// Reply interaction helpers (for tx participation)
// ──────────────────────────────────────────

// FindByIDTx loads a reply inside a transaction.
func (r *ReplyRepository) FindByIDTx(tx *gorm.DB, replyID int) (*model.TopicReply, error) {
	var reply model.TopicReply
	err := tx.First(&reply, replyID).Error
	return &reply, err
}

// FindReplyLike returns an existing reply-like row (or ErrRecordNotFound).
func (r *ReplyRepository) FindReplyLike(tx *gorm.DB, uid, replyID int) (*model.TopicReplyLike, error) {
	var existing model.TopicReplyLike
	err := tx.Where("user_id = ? AND topic_reply_id = ?", uid, replyID).First(&existing).Error
	return &existing, err
}

// CreateReplyLike inserts a new reply-like row.
func (r *ReplyRepository) CreateReplyLike(tx *gorm.DB, uid, replyID int) error {
	return tx.Create(&model.TopicReplyLike{UserID: uid, TopicReplyID: replyID}).Error
}

// DeleteReplyLike removes a previously fetched reply-like row.
func (r *ReplyRepository) DeleteReplyLike(tx *gorm.DB, like *model.TopicReplyLike) error {
	return tx.Delete(like).Error
}

// FindReplyDislike returns an existing reply-dislike row (or ErrRecordNotFound).
func (r *ReplyRepository) FindReplyDislike(tx *gorm.DB, uid, replyID int) (*model.TopicReplyDislike, error) {
	var existing model.TopicReplyDislike
	err := tx.Where("user_id = ? AND topic_reply_id = ?", uid, replyID).First(&existing).Error
	return &existing, err
}

// CreateReplyDislike inserts a new reply-dislike row.
func (r *ReplyRepository) CreateReplyDislike(tx *gorm.DB, uid, replyID int) error {
	return tx.Create(&model.TopicReplyDislike{UserID: uid, TopicReplyID: replyID}).Error
}

// DeleteReplyDislike removes a previously fetched reply-dislike row.
func (r *ReplyRepository) DeleteReplyDislike(tx *gorm.DB, dislike *model.TopicReplyDislike) error {
	return tx.Delete(dislike).Error
}

// AdjustReplyLikeCount adjusts the topic_reply.like_count by `delta`.
func (r *ReplyRepository) AdjustReplyLikeCount(tx *gorm.DB, replyID, delta int) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// AdjustReplyDislikeCount adjusts the topic_reply.dislike_count by `delta`.
func (r *ReplyRepository) AdjustReplyDislikeCount(tx *gorm.DB, replyID, delta int) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).
		Update("dislike_count", gorm.Expr("dislike_count + ?", delta)).Error
}

// ──────────────────────────────────────────
// Reply CRUD helpers (tx-aware)
// ──────────────────────────────────────────

// CreateReply inserts a TopicReply inside the caller tx.
func (r *ReplyRepository) CreateReply(tx *gorm.DB, reply *model.TopicReply) error {
	return tx.Create(reply).Error
}

// CreateReplyTarget inserts a TopicReplyTarget inside the caller tx.
func (r *ReplyRepository) CreateReplyTarget(tx *gorm.DB, target *model.TopicReplyTarget) error {
	return tx.Create(target).Error
}

// UpdateReplyContent updates content + edited timestamp for a reply.
func (r *ReplyRepository) UpdateReplyContent(tx *gorm.DB, replyID int, fields map[string]any) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).Updates(fields).Error
}

// DeleteReplyTargetsByReplyID removes all targets authored by this reply.
func (r *ReplyRepository) DeleteReplyTargetsByReplyID(tx *gorm.DB, replyID int) error {
	return tx.Where("reply_id = ?", replyID).Delete(&model.TopicReplyTarget{}).Error
}

// FindTargetReplyUserID returns the user_id of the target reply (used to
// collect distinct notification recipients when creating a reply).
func (r *ReplyRepository) FindTargetReplyUserID(tx *gorm.DB, replyID int) (int, error) {
	var reply model.TopicReply
	if err := tx.Select("user_id").First(&reply, replyID).Error; err != nil {
		return 0, err
	}
	return reply.UserID, nil
}

