package service

import (
	"net/url"

	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type WebsiteService struct {
	websiteRepo  *repository.WebsiteRepository
	categoryRepo *repository.CategoryRepository
	tagRepo      *repository.TagRepository
	commentRepo  *repository.CommentRepository
}

func NewWebsiteService(
	websiteRepo *repository.WebsiteRepository,
	categoryRepo *repository.CategoryRepository,
	tagRepo *repository.TagRepository,
	commentRepo *repository.CommentRepository,
) *WebsiteService {
	return &WebsiteService{
		websiteRepo:  websiteRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		commentRepo:  commentRepo,
	}
}

// ──────────────────────────────────────────
// GetList — GET /website
// ──────────────────────────────────────────

func (s *WebsiteService) GetList() []dto.WebsiteCard {
	rows := s.websiteRepo.FindAll()
	catMap := s.categoryRepo.FindNamesByIDs(collectCategoryIDs(rows))
	levelMap := s.tagRepo.LevelSumsAll()
	return websiteCardsFromRows(rows, catMap, levelMap)
}

// ──────────────────────────────────────────
// Create — POST /website
// ──────────────────────────────────────────

func (s *WebsiteService) Create(userID int, req *dto.CreateWebsiteRequest) *errors.AppError {
	// Domain parse is left in place for parity with the old handler (unused).
	_, _ = url.Parse(req.URL)

	txErr := s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		website := model.GalgameWebsite{
			Name:        req.Name,
			URL:         req.URL,
			Description: req.Description,
			Icon:        req.Icon,
			Language:    req.Language,
			AgeLimit:    req.AgeLimit,
			CategoryID:  req.CategoryID,
			UserID:      userID,
		}
		if err := s.websiteRepo.Create(tx, &website); err != nil {
			return err
		}
		s.tagRepo.InsertWebsiteTagRelations(tx, website.ID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("创建网站失败")
	}
	return nil
}

// ──────────────────────────────────────────
// GetDetail — GET /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) GetDetail(
	domain string,
	currentUserID int,
) (*dto.WebsiteDetailResponse, *errors.AppError) {
	website, err := s.websiteRepo.FindByDomain(domain)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该网站")
	}

	go s.websiteRepo.IncrementView(website.ID)

	category, _ := s.categoryRepo.FindByID(website.CategoryID)
	catBrief := dto.WebsiteCategoryBrief{}
	if category != nil {
		catBrief = dto.WebsiteCategoryBrief{
			ID:          category.ID,
			Name:        category.Name,
			Label:       category.Label,
			Description: category.Description,
		}
	}

	rels := s.tagRepo.FindRelationsByWebsiteWithTag(website.ID)
	tags := make([]dto.WebsiteTagBrief, len(rels))
	for i, tr := range rels {
		tags[i] = dto.WebsiteTagBrief{
			ID:          tr.Tag.ID,
			Name:        tr.Tag.Name,
			Description: tr.Tag.Description,
			Label:       tr.Tag.Label,
			Level:       tr.Tag.Level,
		}
	}

	detailComments := s.commentRepo.FindByWebsiteForDetail(website.ID)
	commentList := make([]dto.WebsiteDetailComment, len(detailComments))
	for i, cm := range detailComments {
		commentList[i] = dto.WebsiteDetailComment{
			ID:      cm.ID,
			Content: cm.Content,
			User: dto.UserBriefCompact{
				ID: cm.UserID, Name: cm.UserName, Avatar: cm.UserAvatar,
			},
			Created: cm.Created,
			Updated: cm.Updated,
		}
	}

	isLiked, isFavorited := false, false
	if currentUserID > 0 {
		isLiked = s.websiteRepo.HasLike(currentUserID, website.ID)
		isFavorited = s.websiteRepo.HasFavorite(currentUserID, website.ID)
	}

	return &dto.WebsiteDetailResponse{
		ID:            website.ID,
		Name:          website.Name,
		URL:           website.URL,
		Description:   website.Description,
		Icon:          website.Icon,
		View:          website.View,
		Language:      website.Language,
		AgeLimit:      website.AgeLimit,
		Category:      catBrief,
		Tags:          tags,
		LikeCount:     website.LikeCount,
		IsLiked:       isLiked,
		FavoriteCount: website.FavoriteCount,
		IsFavorited:   isFavorited,
		Domain:        website.Domain,
		CreateTime:    website.CreateTime,
		Comment:       commentList,
		Created:       website.CreatedAt,
		Updated:       website.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────
// Update — PUT /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) Update(req *dto.UpdateWebsiteRequest) *errors.AppError {
	txErr := s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.websiteRepo.UpdateFields(tx, req.WebsiteID, map[string]any{
			"name":        req.Name,
			"url":         req.URL,
			"description": req.Description,
			"icon":        req.Icon,
			"category_id": req.CategoryID,
			"age_limit":   req.AgeLimit,
			"language":    req.Language,
		})
		s.tagRepo.ReplaceWebsiteTagRelations(tx, req.WebsiteID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新网站失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete — DELETE /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) Delete(websiteID int) *errors.AppError {
	s.websiteRepo.DeleteByID(websiteID)
	return nil
}

// ──────────────────────────────────────────
// Interactions — PUT /website/:domain/{like,favorite}
// ──────────────────────────────────────────

func (s *WebsiteService) ToggleLike(userID, websiteID int) *errors.AppError {
	s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, err := s.websiteRepo.FindLike(tx, userID, websiteID)
		if err == gorm.ErrRecordNotFound {
			s.websiteRepo.CreateLike(tx, userID, websiteID)
			s.websiteRepo.AdjustLikeCount(tx, websiteID, 1)
		} else if err == nil && existing != nil {
			s.websiteRepo.DeleteLike(tx, existing)
			s.websiteRepo.AdjustLikeCount(tx, websiteID, -1)
		}
		return nil
	})
	return nil
}

func (s *WebsiteService) ToggleFavorite(userID, websiteID int) *errors.AppError {
	s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, err := s.websiteRepo.FindFavorite(tx, userID, websiteID)
		if err == gorm.ErrRecordNotFound {
			s.websiteRepo.CreateFavorite(tx, userID, websiteID)
			s.websiteRepo.AdjustFavoriteCount(tx, websiteID, 1)
		} else if err == nil && existing != nil {
			s.websiteRepo.DeleteFavorite(tx, existing)
			s.websiteRepo.AdjustFavoriteCount(tx, websiteID, -1)
		}
		return nil
	})
	return nil
}
