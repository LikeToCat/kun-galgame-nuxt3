package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

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
		Select(`topic_reply.*,
			"user".name AS user_name,
			"user".avatar AS user_avatar,
			"user".moemoepoint AS user_moemoepoint`).
		Joins(`LEFT JOIN "user" ON "user".id = topic_reply.user_id`).
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
		Select(`topic_reply.*,
			"user".name AS user_name,
			"user".avatar AS user_avatar,
			"user".moemoepoint AS user_moemoepoint`).
		Joins(`LEFT JOIN "user" ON "user".id = topic_reply.user_id`).
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
			tr.user_id AS target_user_id,
			"user".name AS target_user_name,
			"user".avatar AS target_user_avatar`).
		Joins("LEFT JOIN topic_reply tr ON tr.id = topic_reply_target.target_reply_id").
		Joins(`LEFT JOIN "user" ON "user".id = tr.user_id`).
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
	return r.findInteractionStatus("topic_reply_like", "topic_reply_id", uid, replyIDs)
}

func (r *ReplyRepository) FindReplyDislikeStatus(uid int, replyIDs []int) (map[int]bool, error) {
	return r.findInteractionStatus("topic_reply_dislike", "topic_reply_id", uid, replyIDs)
}

func (r *ReplyRepository) findInteractionStatus(table, fkCol string, uid int, ids []int) (map[int]bool, error) {
	if len(ids) == 0 || uid == 0 {
		return make(map[int]bool), nil
	}
	var foundIDs []int
	err := r.db.Table(table).
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

// ────────────────────────────��─────────────
// Comments (batch by reply IDs)
// ──────────────────────────────────────────

type CommentRow struct {
	ID             int
	TopicReplyID   int
	TopicID        int
	Content        string
	UserID         int
	UserName       string
	UserAvatar     string
	TargetUserID   int
	TargetUserName string
	TargetAvatar   string
	LikeCount      int
	CreatedAt      string
}

func (r *ReplyRepository) FindCommentsByReplyIDs(replyIDs []int) (map[int][]CommentRow, error) {
	if len(replyIDs) == 0 {
		return make(map[int][]CommentRow), nil
	}
	var rows []CommentRow
	err := r.db.Table("topic_comment tc").
		Select(`tc.id, tc.topic_reply_id, tc.topic_id, tc.content,
			tc.user_id, u1.name AS user_name, u1.avatar AS user_avatar,
			tc.target_user_id, u2.name AS target_user_name, u2.avatar AS target_avatar,
			(SELECT COUNT(*) FROM topic_comment_like WHERE topic_comment_id = tc.id) AS like_count,
			tc.created AS created_at`).
		Joins(`LEFT JOIN "user" u1 ON u1.id = tc.user_id`).
		Joins(`LEFT JOIN "user" u2 ON u2.id = tc.target_user_id`).
		Where("tc.topic_reply_id IN ?", replyIDs).
		Order("tc.created ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]CommentRow)
	for _, row := range rows {
		result[row.TopicReplyID] = append(result[row.TopicReplyID], row)
	}
	return result, nil
}

func (r *ReplyRepository) FindCommentLikeStatus(uid int, commentIDs []int) (map[int]bool, error) {
	return r.findInteractionStatus("topic_comment_like", "topic_comment_id", uid, commentIDs)
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
// Comment CRUD
// ──────────────────────────────────────────

func (r *ReplyRepository) FindCommentByID(id int) (*model.TopicComment, error) {
	var comment model.TopicComment
	err := r.db.First(&comment, id).Error
	return &comment, err
}

func (r *ReplyRepository) CountCommentLikes(commentID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicCommentLike{}).Where("topic_comment_id = ?", commentID).Count(&count).Error
	return count, err
}
