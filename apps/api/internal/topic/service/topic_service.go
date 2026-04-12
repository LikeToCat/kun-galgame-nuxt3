package service

import (
	"context"
	"fmt"
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
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TopicService struct {
	topicRepo *repository.TopicRepository
	rdb       *redis.Client
}

func NewTopicService(
	topicRepo *repository.TopicRepository,
	rdb *redis.Client,
) *TopicService {
	return &TopicService{topicRepo: topicRepo, rdb: rdb}
}

// ──────────────────────────────────────────
// List
// ──────────��───────────────────────────────

func (s *TopicService) GetList(
	ctx context.Context,
	req *dto.ListTopicsRequest,
	isNSFW bool,
) ([]dto.TopicCard, int64, *errors.AppError) {
	rows, total, err := s.topicRepo.FindList(
		req.Page, req.Limit,
		req.SortField, req.SortOrder, req.Category,
		isNSFW,
	)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取话题列表失败")
	}

	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	tagMap, _ := s.topicRepo.FindTagNamesByTopicIDs(topicIDs)
	sectionMap, _ := s.topicRepo.FindSectionNamesByTopicIDs(topicIDs)

	cards := make([]dto.TopicCard, len(rows))
	for i, r := range rows {
		tags := tagMap[r.ID]
		if tags == nil {
			tags = []string{}
		}
		sections := sectionMap[r.ID]
		if sections == nil {
			sections = []string{}
		}

		hasPoll := false // checked per-item only on detail page

		cards[i] = dto.TopicCard{
			ID:    r.ID,
			Title: r.Title,
			View:  r.View,
			Tags:  tags,
			Sections: sections,
			User: dto.KunUser{
				ID:     r.UserID,
				Name:   r.UserName,
				Avatar: r.UserAvatar,
			},
			Status:           r.Status,
			HasBestAnswer:    r.BestAnswerID != nil,
			IsPollTopic:      hasPoll,
			IsNSFW:           r.IsNSFW,
			LikeCount:        r.LikeCount,
			ReplyCount:       r.ReplyCount,
			CommentCount:     r.CommentCount,
			StatusUpdateTime: r.StatusUpdateTime,
			UpvoteTime:       r.UpvoteTime,
		}
	}

	return cards, total, nil
}

// ──────────────────────────────────────────
// Detail
// ──────────────────────────────────────────

func (s *TopicService) GetDetail(
	ctx context.Context,
	topicID int,
	userInfo *middleware.UserInfo,
) (*dto.TopicDetail, *errors.AppError) {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该话题")
	}

	g, _ := errgroup.WithContext(ctx)

	var author userModel.User
	var tags []string
	var sections []string
	var hasPoll bool
	var isLiked, isDisliked, isFavorited, isUpvoted bool

	g.Go(func() error {
		return s.topicRepo.DB().Table(`"user"`).
			Where("id = ?", topic.UserID).First(&author).Error
	})
	g.Go(func() error {
		var e error
		tags, e = s.topicRepo.FindTagNamesByTopicID(topicID)
		return e
	})
	g.Go(func() error {
		var e error
		sections, e = s.topicRepo.FindSectionNamesByTopicID(topicID)
		return e
	})
	g.Go(func() error {
		var e error
		hasPoll, e = s.topicRepo.HasPoll(topicID)
		return e
	})

	if userInfo != nil {
		uid := userInfo.UID
		g.Go(func() error {
			isLiked, _ = s.topicRepo.HasUserLiked(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isDisliked, _ = s.topicRepo.HasUserDisliked(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isFavorited, _ = s.topicRepo.HasUserFavorited(uid, topicID)
			return nil
		})
		g.Go(func() error {
			isUpvoted, _ = s.topicRepo.HasUserUpvoted(uid, topicID)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, errors.ErrInternal("获取话题详情失败")
	}

	// Increment view asynchronously
	go s.topicRepo.IncrementView(topicID)

	if tags == nil {
		tags = []string{}
	}
	if sections == nil {
		sections = []string{}
	}

	detail := &dto.TopicDetail{
		ID:          topic.ID,
		Title:       topic.Title,
		Content:     topic.Content,
		ContentHtml: markdown.Render(topic.Content),
		View:        topic.View,
		Status:      topic.Status,
		IsNSFW:   topic.IsNSFW,
		Category: topic.Category,
		Sections: sections,
		Tags:     tags,
		User: dto.KunUserWithMoemoepoint{
			ID:          author.ID,
			Name:        author.Name,
			Avatar:      author.Avatar,
			Moemoepoint: author.Moemoepoint,
		},
		LikeCount:        topic.LikeCount,
		IsLiked:          isLiked,
		DislikeCount:     topic.DislikeCount,
		IsDisliked:       isDisliked,
		FavoriteCount:    topic.FavoriteCount,
		IsFavorited:      isFavorited,
		UpvoteCount:      topic.UpvoteCount,
		IsUpvoted:        isUpvoted,
		ReplyCount:       topic.ReplyCount,
		IsPollTopic:      hasPoll,
		StatusUpdateTime: topic.StatusUpdateTime,
		UpvoteTime:       topic.UpvoteTime,
		Edited:           topic.Edited,
		Created:          topic.CreatedAt,
	}

	return detail, nil
}

// ──────────────────────────────────────────
// Create — all checks inside transaction
// ──────────────────────────────────────────

func (s *TopicService) Create(
	ctx context.Context,
	uid int,
	req *dto.CreateTopicRequest,
) (int, *errors.AppError) {
	// Determine if any section is a consume-type
	hasConsumeSection := false
	for _, sec := range req.Sections {
		if constants.TopicSectionConsume[sec] {
			hasConsumeSection = true
			break
		}
	}

	var newTopicID int

	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Lock user row to prevent concurrent moemoepoint manipulation
		var user userModel.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, uid).Error; err != nil {
			return err
		}

		// Check daily limit
		todayCount, err := s.topicRepo.CountTodayTopicsByUser(tx, uid)
		if err != nil {
			return err
		}
		dailyLimit := int64(user.Moemoepoint/10 + 1)
		if todayCount >= dailyLimit {
			return gorm.ErrInvalidData // will be caught below
		}

		// Check moemoepoint for consume sections
		if hasConsumeSection {
			if user.Moemoepoint < constants.CostConsumeSection {
				return gorm.ErrInvalidData
			}
		}

		// Create topic
		topic := &topicModel.Topic{
			Title:    req.Title,
			Content:  req.Content,
			Category: req.Category,
			IsNSFW:   req.IsNSFW,
			UserID:   uid,
		}
		if err := tx.Create(topic).Error; err != nil {
			return err
		}
		newTopicID = topic.ID

		// Tags — find or create, then create relations
		tags, err := s.topicRepo.FindOrCreateTags(req.Tags)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			rel := topicModel.TopicTagRelation{TopicID: topic.ID, TagID: tag.ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
				return err
			}
		}

		// Sections
		var sections []topicModel.TopicSection
		if err := tx.Where("name IN ?", req.Sections).Find(&sections).Error; err != nil {
			return err
		}
		for _, sec := range sections {
			rel := topicModel.TopicSectionRelation{TopicID: topic.ID, TopicSectionID: sec.ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
				return err
			}
		}

		// Moemoepoint
		pointsDelta := constants.RewardCreateTopic
		if hasConsumeSection {
			pointsDelta = -constants.CostConsumeSection
		}
		if err := tx.Model(&userModel.User{}).Where("id = ?", uid).
			Update("moemoepoint", gorm.Expr("moemoepoint + ?", pointsDelta)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrInvalidData {
			if hasConsumeSection {
				return 0, errors.ErrBadRequest("您的萌萌点不足, 无法发布此类型话题")
			}
			return 0, errors.ErrBadRequest("您今日发布的话题已达上限")
		}
		return 0, errors.ErrInternal("创建话题失败")
	}

	return newTopicID, nil
}

// ──────────────────────────────────────────
// Update
// ──────────────────────────────────────────

func (s *TopicService) Update(
	ctx context.Context,
	uid, role int,
	topicID int,
	req *dto.UpdateTopicRequest,
) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限编辑此话题")
	}

	now := time.Now()
	txErr := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).Updates(map[string]any{
			"title":              req.Title,
			"content":            req.Content,
			"category":           req.Category,
			"is_nsfw":            req.IsNSFW,
			"edited":             &now,
			"status_update_time": now,
		}).Error; err != nil {
			return err
		}

		// Replace tags
		tags, err := s.topicRepo.FindOrCreateTags(req.Tags)
		if err != nil {
			return err
		}
		tagIDs := make([]int, len(tags))
		for i, t := range tags {
			tagIDs[i] = t.ID
		}
		if err := s.topicRepo.ReplaceTopicTags(tx, topicID, tagIDs); err != nil {
			return err
		}

		// Replace sections
		var sections []topicModel.TopicSection
		if err := tx.Where("name IN ?", req.Sections).Find(&sections).Error; err != nil {
			return err
		}
		sectionIDs := make([]int, len(sections))
		for i, sec := range sections {
			sectionIDs[i] = sec.ID
		}
		return s.topicRepo.ReplaceSectionRelations(tx, topicID, sectionIDs)
	})

	if txErr != nil {
		return errors.ErrInternal("更新话题失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Interactions — all checks inside transaction
// ──────────────────────────────────────────

func (s *TopicService) ToggleLike(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		var topic topicModel.Topic
		if err := tx.First(&topic, topicID).Error; err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}
		if topic.UserID == uid {
			return gorm.ErrInvalidData
		}

		var existing topicModel.TopicLike
		result := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Add like
			tx.Create(&topicModel.TopicLike{UserID: uid, TopicID: topicID})
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("like_count", gorm.Expr("like_count + 1"))
			// Reward content owner
			tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
			// Message
			createDedupMessage(tx, uid, topic.UserID, "liked", topicID)
		} else {
			// Remove like
			tx.Delete(&existing)
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("like_count", gorm.Expr("like_count - 1"))
			tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
				Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
		}
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能给自己点赞")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicService) ToggleDislike(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		var topic topicModel.Topic
		if err := tx.First(&topic, topicID).Error; err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}

		var existing topicModel.TopicDislike
		result := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&topicModel.TopicDislike{UserID: uid, TopicID: topicID})
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("dislike_count", gorm.Expr("dislike_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("dislike_count", gorm.Expr("dislike_count - 1"))
		}
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicService) Upvote(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		var topic topicModel.Topic
		if err := tx.First(&topic, topicID).Error; err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}
		if topic.UserID == uid {
			return gorm.ErrInvalidData
		}

		// Check sender balance with FOR UPDATE
		var user userModel.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, uid).Error; err != nil {
			return err
		}
		if user.Moemoepoint < constants.CostUpvoteSender {
			return gorm.ErrCheckConstraintViolated
		}

		now := time.Now()

		// Upvote allows duplicates (no unique constraint)
		tx.Create(&topicModel.TopicUpvote{UserID: uid, TopicID: topicID})
		tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).Updates(map[string]any{
			"upvote_count":       gorm.Expr("upvote_count + 1"),
			"status_update_time": now,
			"upvote_time":        &now,
		})

		// Deduct sender, reward owner
		tx.Model(&userModel.User{}).Where("id = ?", uid).
			Update("moemoepoint", gorm.Expr("moemoepoint - ?", constants.CostUpvoteSender))
		tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
			Update("moemoepoint", gorm.Expr("moemoepoint + ?", constants.RewardUpvoteOwner))

		createDedupMessage(tx, uid, topic.UserID, "upvoted", topicID)
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err == gorm.ErrInvalidData {
		return errors.ErrBadRequest("您不能推自己的话题")
	}
	if err == gorm.ErrCheckConstraintViolated {
		return errors.ErrBadRequest("萌萌点不足, 推话题需要 7 萌萌点")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicService) ToggleFavorite(ctx context.Context, uid, topicID int) *errors.AppError {
	err := s.topicRepo.DB().Transaction(func(tx *gorm.DB) error {
		var topic topicModel.Topic
		if err := tx.First(&topic, topicID).Error; err != nil {
			return err
		}
		if topic.Status == 1 {
			return gorm.ErrRecordNotFound
		}

		var existing topicModel.TopicFavorite
		result := tx.Where("user_id = ? AND topic_id = ?", uid, topicID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&topicModel.TopicFavorite{UserID: uid, TopicID: topicID})
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("favorite_count", gorm.Expr("favorite_count + 1"))
			if uid != topic.UserID {
				tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint + 1"))
				createDedupMessage(tx, uid, topic.UserID, "favorite", topicID)
			}
		} else {
			tx.Delete(&existing)
			tx.Model(&topicModel.Topic{}).Where("id = ?", topicID).
				Update("favorite_count", gorm.Expr("favorite_count - 1"))
			if uid != topic.UserID {
				tx.Model(&userModel.User{}).Where("id = ?", topic.UserID).
					Update("moemoepoint", gorm.Expr("moemoepoint - 1"))
			}
		}
		return nil
	})

	if err == gorm.ErrRecordNotFound {
		return errors.ErrNotFound("未找到该话题")
	}
	if err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicService) ToggleHide(ctx context.Context, uid, role, topicID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid && role < 2 {
		return errors.ErrForbidden("您没有权限操作此话题")
	}

	newStatus := 1
	if topic.Status == 1 {
		newStatus = 0
	}
	if err := s.topicRepo.UpdateFields(topicID, map[string]any{"status": newStatus}); err != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *TopicService) SetBestAnswer(ctx context.Context, uid, topicID, replyID int) *errors.AppError {
	topic, err := s.topicRepo.FindByID(topicID)
	if err != nil {
		return errors.ErrNotFound("未找到该话题")
	}
	if topic.UserID != uid {
		return errors.ErrForbidden("只有话题作者可以设置最佳回答")
	}

	if err := s.topicRepo.UpdateFields(topicID, map[string]any{
		"best_answer_id": &replyID,
	}); err != nil {
		return errors.ErrInternal("设置最佳回答失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Message helper — dedup within transaction
// ──────────────────────────────────────────

func createDedupMessage(tx *gorm.DB, senderID, receiverID int, msgType string, topicID int) {
	if senderID == receiverID {
		return
	}
	link := fmt.Sprintf("/topic/%d", topicID)
	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}
	tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Link:       link,
		Status:     "unread",
	})
}
