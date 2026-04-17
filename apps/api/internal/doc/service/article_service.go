package service

import (
	"time"

	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

type ArticleService struct {
	articleRepo *repository.ArticleRepository
}

func NewArticleService(articleRepo *repository.ArticleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
}

// ──────────────────────────────────────────
// GetList — GET /doc/article
// ──────────────────────────────────────────

// ArticleListResult carries the list + total for paginated handler responses.
type ArticleListResult struct {
	Items []model.DocArticle
	Total int64
}

func (s *ArticleService) GetList(req *dto.GetArticlesRequest) *ArticleListResult {
	if req.OrderBy == "" {
		req.OrderBy = "published_time"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	items, total := s.articleRepo.FindPaginated(req)
	return &ArticleListResult{Items: items, Total: total}
}

// ──────────────────────────────────────────
// GetBySlug — GET /doc/article/:slug
// ──────────────────────────────────────────

func (s *ArticleService) GetBySlug(slug string) (*dto.ArticleDetailResponse, *errors.AppError) {
	article, err := s.articleRepo.FindBySlug(slug)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该文章")
	}

	// Bump view asynchronously to preserve the old fire-and-forget behavior.
	go s.articleRepo.IncrementView(article.ID)

	return &dto.ArticleDetailResponse{
		ID:              article.ID,
		Title:           article.Title,
		Slug:            article.Slug,
		Path:            article.Path,
		Description:     article.Description,
		Banner:          article.Banner,
		Status:          article.Status,
		IsPin:           article.IsPin,
		View:            article.View,
		PublishedTime:   article.PublishedTime,
		EditedTime:      article.EditedTime,
		ContentMarkdown: article.ContentMarkdown,
		ContentHTML:     markdown.Render(article.ContentMarkdown),
		CategoryID:      article.CategoryID,
		AuthorID:        article.AuthorID,
		Created:         article.CreatedAt,
		Updated:         article.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────
// Create — POST /doc/article
// ──────────────────────────────────────────

func (s *ArticleService) Create(uid int, req *dto.CreateArticleRequest) (*model.DocArticle, *errors.AppError) {
	now := time.Now()
	article := model.DocArticle{
		Title:           req.Title,
		Slug:            req.Slug,
		Path:            "/doc/" + req.Slug,
		Description:     req.Description,
		Banner:          req.Banner,
		Status:          req.Status,
		IsPin:           req.IsPin,
		ContentMarkdown: req.ContentMarkdown,
		CategoryID:      req.CategoryID,
		AuthorID:        uid,
		PublishedTime:   now,
	}

	txErr := s.articleRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articleRepo.Create(tx, &article); err != nil {
			return err
		}
		s.articleRepo.InsertTagRelations(tx, article.ID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建文章失败")
	}

	return &article, nil
}

// ──────────────────────────────────────────
// Update — PUT /doc/article
// ──────────────────────────────────────────

func (s *ArticleService) Update(req *dto.UpdateArticleRequest) *errors.AppError {
	now := time.Now()
	updates := map[string]any{
		"title":            req.Title,
		"slug":             req.Slug,
		"path":             "/doc/" + req.Slug,
		"description":      req.Description,
		"banner":           req.Banner,
		"status":           req.Status,
		"is_pin":           req.IsPin,
		"content_markdown": req.ContentMarkdown,
		"category_id":      req.CategoryID,
		"edited_time":      &now,
	}

	txErr := s.articleRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.articleRepo.UpdateFields(tx, req.ArticleID, updates); err != nil {
			return err
		}
		s.articleRepo.ReplaceTagRelations(tx, req.ArticleID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新文章失败")
	}

	return nil
}

// ──────────────────────────────────────────
// Delete — DELETE /doc/article
// ──────────────────────────────────────────

func (s *ArticleService) Delete(articleID int) *errors.AppError {
	s.articleRepo.DeleteTagRelationsByArticleID(articleID)
	s.articleRepo.DeleteByID(articleID)
	return nil
}
