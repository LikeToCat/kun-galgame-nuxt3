package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"

	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReplyService struct {
	replyRepo *repository.ReplyRepository
	topicRepo *repository.TopicRepository
	rdb       *redis.Client
}

func NewReplyService(
	replyRepo *repository.ReplyRepository,
	topicRepo *repository.TopicRepository,
	rdb *redis.Client,
) *ReplyService {
	return &ReplyService{replyRepo: replyRepo, topicRepo: topicRepo, rdb: rdb}
}

// ──────────────────────────────────────────
// List replies
// ──────────────────────────────────────────

func (s *ReplyService) GetReplies(
	ctx context.Context,
	req *dto.ListRepliesRequest,
	userInfo *middleware.UserInfo,
) ([]dto.TopicReplyResponse, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(req.TopicID)
	if err != nil {
		return []dto.TopicReplyResponse{}, nil
	}

	// Collect special reply IDs (pinned + best answer)
	var specialIDs []int
	if topic.PinnedReplyID != nil {
		specialIDs = append(specialIDs, *topic.PinnedReplyID)
	}
	if topic.BestAnswerID != nil && (topic.PinnedReplyID == nil || *topic.BestAnswerID != *topic.PinnedReplyID) {
		specialIDs = append(specialIDs, *topic.BestAnswerID)
	}

	var result []dto.TopicReplyResponse

	// On page 1, prepend special replies
	if req.Page == 1 && len(specialIDs) > 0 {
		specialRows, err := s.replyRepo.FindRepliesByIDs(specialIDs)
		if err == nil {
			specialResponses := s.buildReplyResponses(specialRows, topic, userInfo)
			result = append(result, specialResponses...)
		}
	}

	// Regular replies (exclude special IDs)
	regularRows, err := s.replyRepo.FindRepliesPaginated(
		req.TopicID, specialIDs,
		req.Page, req.Limit, req.SortOrder,
	)
	if err != nil {
		return nil, errors.ErrInternal("获取回复列表失败")
	}

	regularResponses := s.buildReplyResponses(regularRows, topic, userInfo)
	result = append(result, regularResponses...)

	if result == nil {
		result = []dto.TopicReplyResponse{}
	}
	return result, nil
}

// ──────────────────────────────────────────
// Reply detail
// ──────────────────────────────────────────

func (s *ReplyService) GetReplyDetail(
	ctx context.Context,
	replyID int,
	userInfo *middleware.UserInfo,
) (*dto.TopicReplyResponse, *errors.AppError) {
	rows, err := s.replyRepo.FindRepliesByIDs([]int{replyID})
	if err != nil || len(rows) == 0 {
		return nil, errors.ErrNotFound("未找到该回复")
	}

	reply := rows[0]
	topic, _ := s.topicRepo.FindByID(reply.TopicID)
	responses := s.buildReplyResponses(rows, topic, userInfo)
	if len(responses) == 0 {
		return nil, errors.ErrNotFound("未找到该回复")
	}
	return &responses[0], nil
}

// ──────────────────────────────────────────
// Create reply — floor calculation inside tx
// ──────────────────────────────────────────

func (s *ReplyService) CreateReply(
	ctx context.Context,
	uid int,
	req *dto.CreateReplyRequest,
) (*dto.TopicReplyResponse, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(req.TopicID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该话题")
	}

	validTargets := make([]dto.ReplyTarget, 0)
	for _, t := range req.Targets {
		if strings.TrimSpace(t.Content) != "" {
			validTargets = append(validTargets, t)
		}
	}

	if strings.TrimSpace(req.Content) == "" && len(validTargets) == 0 {
		return nil, errors.ErrBadRequest("回复内容不能为空")
	}

	var newReply *topicModel.TopicReply

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Floor calculation INSIDE transaction
		maxFloor, err := s.replyRepo.GetMaxFloor(tx, req.TopicID)
		if err != nil {
			return err
		}

		newReply = &topicModel.TopicReply{
			UserID:  uid,
			TopicID: req.TopicID,
			Floor:   maxFloor + 1,
			Content: req.Content,
		}
		if err := tx.Create(newReply).Error; err != nil {
			return err
		}

		// Create targets
		for _, t := range validTargets {
			target := topicModel.TopicReplyTarget{
				ReplyID:       newReply.ID,
				TargetReplyID: t.TargetReplyID,
				Content:       t.Content,
			}
			if err := tx.Create(&target).Error; err != nil {
				return err
			}
		}

		// Update topic status_update_time
		tx.Model(&topicModel.Topic{}).Where("id = ?", req.TopicID).
			Updates(map[string]any{"status_update_time": time.Now()})

		// Batch moemoepoint for target users
		targetUserMap := make(map[int]bool)
		for _, t := range validTargets {
			var targetReply topicModel.TopicReply
			if tx.Select("user_id").First(&targetReply, t.TargetReplyID).Error == nil {
				if targetReply.UserID != uid && !targetUserMap[targetReply.UserID] {
					targetUserMap[targetReply.UserID] = true
				}
			}
		}
		if len(targetUserMap) > 0 {
			targetUIDs := make([]int, 0, len(targetUserMap))
			for id := range targetUserMap {
				targetUIDs = append(targetUIDs, id)
			}
			tx.Model(&userModel.User{}).Where("id IN ?", targetUIDs).
				Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardReply))

			// Batch create messages
			link := fmt.Sprintf("/topic/%d", req.TopicID)
			preview := truncate(req.Content, constants.TextPreviewLength)
			for _, targetUID := range targetUIDs {
				tx.Create(&msgModel.Message{
					SenderID:   uid,
					ReceiverID: targetUID,
					Type:       "replied",
					Content:    preview,
					Link:       link,
					Status:     "unread",
				})
			}
		}

		// Reward topic owner
		if strings.TrimSpace(req.Content) != "" && topic.UserID != uid {
			tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardReply))

			link := fmt.Sprintf("/topic/%d", req.TopicID)
			preview := truncate(req.Content, constants.TextPreviewLength)
			tx.Create(&msgModel.Message{
				SenderID:   uid,
				ReceiverID: topic.UserID,
				Type:       "replied",
				Content:    preview,
				Link:       link,
				Status:     "unread",
			})
		}

		return nil
	})

	if txErr != nil {
		return nil, errors.ErrInternal("创建回复失败")
	}

	// Build response outside transaction
	rows, _ := s.replyRepo.FindRepliesByIDs([]int{newReply.ID})
	if len(rows) == 0 {
		return nil, errors.ErrInternal("创建回复失败")
	}
	responses := s.buildReplyResponses(rows, topic, nil)
	return &responses[0], nil
}

// ──────────────────────────────────────────
// Update reply
// ──────────────────────────────────────────

func (s *ReplyService) UpdateReply(
	ctx context.Context,
	uid int,
	req *dto.UpdateReplyRequest,
) *errors.AppError {
	reply, err := s.replyRepo.FindByID(req.ReplyID)
	if err != nil {
		return errors.ErrNotFound("未找到该回复")
	}
	if reply.UserID != uid {
		return errors.ErrForbidden("您没有权限编辑此回复")
	}

	now := time.Now()
	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		tx.Model(&topicModel.TopicReply{}).Where("id = ?", req.ReplyID).
			Updates(map[string]any{"content": req.Content, "edited": &now})

		// Replace targets
		if len(req.Targets) > 0 {
			tx.Where("reply_id = ?", req.ReplyID).Delete(&topicModel.TopicReplyTarget{})
			for _, t := range req.Targets {
				if strings.TrimSpace(t.Content) == "" {
					continue
				}
				tx.Create(&topicModel.TopicReplyTarget{
					ReplyID:       req.ReplyID,
					TargetReplyID: t.TargetReplyID,
					Content:       t.Content,
				})
			}
		}

		return nil
	})

	if txErr != nil {
		return errors.ErrInternal("更新回复失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete reply — cascade + moemoepoint penalty
// ──────────────────────────────────────────

func (s *ReplyService) DeleteReply(
	ctx context.Context,
	uid, role, replyID int,
) *errors.AppError {
	reply, err := s.replyRepo.FindByID(replyID)
	if err != nil {
		return errors.ErrNotFound("未找到该回复")
	}
	if reply.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限删除此回复")
	}

	commentCount, likeCount, targetCount, targetByCount, _ := s.replyRepo.CountReplyRelated(replyID)

	penalty := 3
	if reply.UserID == uid && role < 2 {
		penalty = 3 * int(commentCount+likeCount+targetCount+targetByCount+1)
	}

	txErr := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Check balance
		var user userModel.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, reply.UserID).Error; err != nil {
			return err
		}
		if user.Moemoepoint < penalty {
			return gorm.ErrCheckConstraintViolated
		}

		// Cascade collect + delete
		allIDs, err := s.replyRepo.CollectCascadeReplyIDs(tx, []int{replyID})
		if err != nil {
			return err
		}
		if err := s.replyRepo.DeleteRepliesByIDs(tx, allIDs); err != nil {
			return err
		}

		// Deduct moemoepoint
		return tx.Model(&userModel.User{}).Where("id = ?", reply.UserID).
			Update("moemoepoint", gorm.Expr("moemoepoint - ?", penalty)).Error
	})

	if txErr == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 无法删除此回复")
	}
	if txErr != nil {
		return errors.ErrInternal("删除回复失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Reply interactions
// ──────────────────────────────────────────

func (s *ReplyService) ToggleReplyLike(ctx context.Context, uid, replyID int) *errors.AppError {
	err := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		var reply topicModel.TopicReply
		if err := tx.First(&reply, replyID).Error; err != nil {
			return err
		}
		if reply.UserID == uid {
			return gorm.ErrInvalidData
		}

		var existing topicModel.TopicReplyLike
		result := tx.Where("user_id = ? AND topic_reply_id = ?", uid, replyID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&topicModel.TopicReplyLike{UserID: uid, TopicReplyID: replyID})
			tx.Model(&topicModel.TopicReply{}).Where("id = ?", replyID).
				Update("like_count", gorm.Expr("like_count + 1"))
			tx.Model(&userModel.User{}).Where("id = ?", reply.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + 1"))

			link := fmt.Sprintf("/topic/%d", reply.TopicID)
			preview := truncate(reply.Content, constants.TextPreviewLength)
			createDedupMessageByLink(tx, uid, reply.UserID, "liked", preview, link)
		} else {
			tx.Delete(&existing)
			tx.Model(&topicModel.TopicReply{}).Where("id = ?", replyID).
				Update("like_count", gorm.Expr("like_count - 1"))
			tx.Model(&userModel.User{}).Where("id = ?", reply.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
		}
		return nil
	})

	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能给自己的回复点赞")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *ReplyService) ToggleReplyDislike(ctx context.Context, uid, replyID int) *errors.AppError {
	err := s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		var reply topicModel.TopicReply
		if err := tx.First(&reply, replyID).Error; err != nil {
			return err
		}
		if reply.UserID == uid {
			return gorm.ErrInvalidData
		}

		var existing topicModel.TopicReplyDislike
		result := tx.Where("user_id = ? AND topic_reply_id = ?", uid, replyID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&topicModel.TopicReplyDislike{UserID: uid, TopicReplyID: replyID})
			tx.Model(&topicModel.TopicReply{}).Where("id = ?", replyID).
				Update("dislike_count", gorm.Expr("dislike_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&topicModel.TopicReply{}).Where("id = ?", replyID).
				Update("dislike_count", gorm.Expr("dislike_count - 1"))
		}
		return nil
	})

	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能踩自己的回复")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *ReplyService) PinReply(ctx context.Context, uid, role, topicID, replyID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限置顶回复")
	}

	var newPinned *int
	if topic.PinnedReplyID != nil && *topic.PinnedReplyID == replyID {
		newPinned = nil // unpin
	} else {
		newPinned = &replyID
	}

	if err := s.topicRepo.UpdateFields(topicID, map[string]any{"pinned_reply_id": newPinned}); err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────

func (s *ReplyService) buildReplyResponses(
	rows []repository.ReplyRow,
	topic *topicModel.Topic,
	userInfo *middleware.UserInfo,
) []dto.TopicReplyResponse {
	if len(rows) == 0 {
		return nil
	}

	replyIDs := make([]int, len(rows))
	for i, r := range rows {
		replyIDs[i] = r.ID
	}

	targetMap, _ := s.replyRepo.FindTargetsByReplyIDs(replyIDs)
	commentMap, _ := s.replyRepo.FindCommentsByReplyIDs(replyIDs)

	var likeMap, dislikeMap map[int]bool
	var commentLikeMap map[int]bool
	if userInfo != nil {
		likeMap, _ = s.replyRepo.FindReplyLikeStatus(userInfo.UID, replyIDs)
		dislikeMap, _ = s.replyRepo.FindReplyDislikeStatus(userInfo.UID, replyIDs)

		// Collect comment IDs for like status
		var commentIDs []int
		for _, comments := range commentMap {
			for _, c := range comments {
				commentIDs = append(commentIDs, c.ID)
			}
		}
		commentLikeMap, _ = s.replyRepo.FindCommentLikeStatus(userInfo.UID, commentIDs)
	}

	responses := make([]dto.TopicReplyResponse, len(rows))
	for i, r := range rows {
		// Build targets
		var targets []dto.ReplyTargetResponse
		if ts, ok := targetMap[r.ID]; ok {
			for _, t := range ts {
				preview := truncate(t.TargetContent, 150)
				targets = append(targets, dto.ReplyTargetResponse{
					ID:                   t.TargetReplyID,
					Floor:                t.TargetFloor,
					User:                 dto.KunUser{ID: t.TargetUserID, Name: t.TargetUserName, Avatar: t.TargetUserAvatar},
					ContentPreview:       preview,
					ReplyContentMarkdown: t.Content,
					ReplyContentHtml:     markdown.Render(t.Content),
				})
			}
		}
		if targets == nil {
			targets = []dto.ReplyTargetResponse{}
		}

		// Build comments
		var comments []dto.TopicCommentResponse
		if cs, ok := commentMap[r.ID]; ok {
			for _, c := range cs {
				isLiked := false
				if commentLikeMap != nil {
					isLiked = commentLikeMap[c.ID]
				}
				comments = append(comments, dto.TopicCommentResponse{
					ID:         c.ID,
					ReplyID:    c.TopicReplyID,
					TopicID:    c.TopicID,
					User:       dto.KunUser{ID: c.UserID, Name: c.UserName, Avatar: c.UserAvatar},
					TargetUser: dto.KunUser{ID: c.TargetUserID, Name: c.TargetUserName, Avatar: c.TargetAvatar},
					Content:    c.Content,
					IsLiked:    isLiked,
					LikeCount:  c.LikeCount,
				})
			}
		}
		if comments == nil {
			comments = []dto.TopicCommentResponse{}
		}

		isPinned := topic != nil && topic.PinnedReplyID != nil && *topic.PinnedReplyID == r.ID
		isBestAnswer := topic != nil && topic.BestAnswerID != nil && *topic.BestAnswerID == r.ID

		responses[i] = dto.TopicReplyResponse{
			ID:      r.ID,
			TopicID: r.TopicID,
			Floor:   r.Floor,
			User: dto.KunUserWithMoemoepoint{
				ID: r.UserID, Name: r.UserName,
				Avatar: r.UserAvatar, Moemoepoint: r.UserMoemoepoint,
			},
			Edited:          r.Edited,
			ContentMarkdown: r.Content,
			ContentHtml:     markdown.Render(r.Content),
			LikeCount:       r.LikeCount,
			IsLiked:         likeMap[r.ID],
			DislikeCount:    r.DislikeCount,
			IsDisliked:      dislikeMap[r.ID],
			Comments:        comments,
			Targets:         targets,
			IsPinned:        isPinned,
			IsBestAnswer:    isBestAnswer,
			Created:         r.CreatedAt,
		}
	}
	return responses
}

func createDedupMessageByLink(tx *gorm.DB, senderID, receiverID int, msgType, content, link string) {
	if senderID == receiverID {
		return
	}
	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}
	tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: content, Link: link, Status: "unread",
	})
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
