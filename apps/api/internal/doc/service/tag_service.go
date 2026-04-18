package service

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/pkg/errors"
)

type TagService struct {
	tagRepo *repository.TagRepository
}

func NewTagService(tagRepo *repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

// TagListResult carries the list + total for paginated handler responses.
type TagListResult struct {
	Items []model.DocTag
	Total int64
}

// GetList — GET /doc/tag
func (s *TagService) GetList(req *dto.GetTagsRequest) *TagListResult {
	items, total := s.tagRepo.FindPaginated(req.Keyword, req.Page, req.Limit)
	return &TagListResult{Items: items, Total: total}
}

// Create — POST /doc/tag
func (s *TagService) Create(tag *model.DocTag) *errors.AppError {
	if err := s.tagRepo.Create(tag); err != nil {
		return errors.ErrInternal("创建标签失败")
	}
	return nil
}

// Delete — DELETE /doc/tag
func (s *TagService) Delete(tagID int) *errors.AppError {
	s.tagRepo.DeleteByID(tagID)
	return nil
}

// Update — PUT /doc/tag (admin role >= 2 enforced at the route level).
func (s *TagService) Update(req *dto.UpdateTagRequest) (*model.DocTag, *errors.AppError) {
	tag, err := s.tagRepo.Update(req.TagID, map[string]any{
		"slug":        req.Slug,
		"title":       req.Title,
		"description": req.Description,
	})
	if err != nil {
		return nil, errors.ErrInternal("更新标签失败")
	}
	return tag, nil
}
