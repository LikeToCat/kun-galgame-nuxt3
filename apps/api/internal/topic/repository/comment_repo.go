package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

// CommentRepository owns TopicComment rows (comments attached to a reply):
// CRUD, like interactions + counts, and the batch lookup used when building
// a reply's comment list.
//
// It piggy-backs on ReplyRepository.LockUserForUpdate via the parent service —
// a user row lock is still taken through the reply repo (see CommentService).
type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Comment rows attached to replies (batch fetch)
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

func (r *CommentRepository) FindCommentsByReplyIDs(replyIDs []int) (map[int][]CommentRow, error) {
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

// FindCommentLikeStatus reports which of the given comment IDs the user has liked.
func (r *CommentRepository) FindCommentLikeStatus(uid int, commentIDs []int) (map[int]bool, error) {
	return findInteractionStatus(r.db, "topic_comment_like", "topic_comment_id", uid, commentIDs)
}

// ──────────────────────────────────────────
// Comment CRUD
// ──────────────────────────────────────────

func (r *CommentRepository) FindCommentByID(id int) (*model.TopicComment, error) {
	var comment model.TopicComment
	err := r.db.First(&comment, id).Error
	return &comment, err
}

func (r *CommentRepository) CountCommentLikes(commentID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicCommentLike{}).Where("topic_comment_id = ?", commentID).Count(&count).Error
	return count, err
}

// CreateComment inserts a TopicComment inside the caller tx.
func (r *CommentRepository) CreateComment(tx *gorm.DB, c *model.TopicComment) error {
	return tx.Create(c).Error
}

// FindCommentByIDTx loads a TopicComment inside a transaction.
func (r *CommentRepository) FindCommentByIDTx(tx *gorm.DB, commentID int) (*model.TopicComment, error) {
	var comment model.TopicComment
	err := tx.First(&comment, commentID).Error
	return &comment, err
}

// FindCommentLike looks up an existing comment like row.
func (r *CommentRepository) FindCommentLike(tx *gorm.DB, uid, commentID int) (*model.TopicCommentLike, error) {
	var existing model.TopicCommentLike
	err := tx.Where("user_id = ? AND topic_comment_id = ?", uid, commentID).First(&existing).Error
	return &existing, err
}

// CreateCommentLike inserts a comment like row.
func (r *CommentRepository) CreateCommentLike(tx *gorm.DB, uid, commentID int) error {
	return tx.Create(&model.TopicCommentLike{UserID: uid, TopicCommentID: commentID}).Error
}

// DeleteCommentLike removes a previously fetched comment like row.
func (r *CommentRepository) DeleteCommentLike(tx *gorm.DB, like *model.TopicCommentLike) error {
	return tx.Delete(like).Error
}

// DeleteCommentLikesForComment removes all likes targeting a given comment.
func (r *CommentRepository) DeleteCommentLikesForComment(tx *gorm.DB, commentID int) error {
	return tx.Where("topic_comment_id = ?", commentID).Delete(&model.TopicCommentLike{}).Error
}

// DeleteCommentByID removes a single TopicComment row by primary key.
func (r *CommentRepository) DeleteCommentByID(tx *gorm.DB, commentID int) error {
	return tx.Delete(&model.TopicComment{}, commentID).Error
}
