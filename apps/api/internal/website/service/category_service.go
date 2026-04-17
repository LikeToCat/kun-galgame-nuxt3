package service

import (
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/errors"
)

type CategoryService struct {
	categoryRepo *repository.CategoryRepository
	websiteRepo  *repository.WebsiteRepository
	tagRepo      *repository.TagRepository
}

func NewCategoryService(
	categoryRepo *repository.CategoryRepository,
	websiteRepo *repository.WebsiteRepository,
	tagRepo *repository.TagRepository,
) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		websiteRepo:  websiteRepo,
		tagRepo:      tagRepo,
	}
}

// ──────────────────────────────────────────
// GetDetail — GET /website-category/:name
// ──────────────────────────────────────────

func (s *CategoryService) GetDetail(name string) (*dto.WebsiteCategoryDetailResponse, *errors.AppError) {
	category, err := s.categoryRepo.FindByName(name)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该分类")
	}

	rows := s.websiteRepo.FindByCategoryID(category.ID)
	websiteIDs := collectWebsiteIDs(rows)
	levelMap := s.tagRepo.LevelSumsByWebsiteIDs(websiteIDs)
	cards := websiteCardsFromRowsSingleCategory(rows, category.Name, levelMap)

	return &dto.WebsiteCategoryDetailResponse{
		ID:           category.ID,
		Name:         category.Name,
		Label:        category.Label,
		Description:  category.Description,
		WebsiteCount: len(rows),
		Websites:     cards,
		Created:      category.CreatedAt,
		Updated:      category.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────
// Update — PUT /website-category
// ──────────────────────────────────────────

func (s *CategoryService) Update(req *dto.UpdateWebsiteCategoryRequest) *errors.AppError {
	s.categoryRepo.UpdateFields(req.CategoryID, map[string]any{
		"name":        req.Name,
		"label":       req.Label,
		"description": req.Description,
	})
	return nil
}
