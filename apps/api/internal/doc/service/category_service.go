package service

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/pkg/errors"
)

type CategoryService struct {
	categoryRepo *repository.CategoryRepository
}

func NewCategoryService(categoryRepo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

// CategoryListResult carries the list + total for paginated handler responses.
type CategoryListResult struct {
	Items []model.DocCategory
	Total int64
}

// GetList — GET /doc/category
func (s *CategoryService) GetList(req *dto.GetCategoriesRequest) *CategoryListResult {
	items, total := s.categoryRepo.FindPaginated(req.Keyword, req.Page, req.Limit)
	return &CategoryListResult{Items: items, Total: total}
}

// Create — POST /doc/category
func (s *CategoryService) Create(category *model.DocCategory) *errors.AppError {
	if err := s.categoryRepo.Create(category); err != nil {
		return errors.ErrInternal("创建分类失败")
	}
	return nil
}

// Update — PUT /doc/category
func (s *CategoryService) Update(req *dto.UpdateCategoryRequest) *errors.AppError {
	s.categoryRepo.UpdateFields(req.CategoryID, map[string]any{
		"slug":        req.Slug,
		"title":       req.Title,
		"description": req.Description,
		"icon":        req.Icon,
		"sort_order":  req.SortOrder,
	})
	return nil
}

// Delete — DELETE /doc/category
func (s *CategoryService) Delete(categoryID int) *errors.AppError {
	s.categoryRepo.DeleteByID(categoryID)
	return nil
}
