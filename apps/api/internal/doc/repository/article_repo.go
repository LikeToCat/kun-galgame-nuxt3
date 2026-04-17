package repository

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// FindPaginated applies the filters from GetArticlesRequest and returns the
// matching articles plus total count.
func (r *ArticleRepository) FindPaginated(req *dto.GetArticlesRequest) ([]model.DocArticle, int64) {
	query := r.db.Model(&model.DocArticle{})

	// Default: only published articles
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	} else {
		query = query.Where("status = 1")
	}
	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}
	if req.IsPin != nil {
		query = query.Where("is_pin = ?", *req.IsPin)
	}
	if req.Keyword != "" {
		query = query.Where(
			"title ILIKE ? OR slug ILIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%",
		)
	}
	if req.TagID != nil {
		query = query.Where(
			"id IN (SELECT doc_article_id FROM doc_article_tag_relation WHERE doc_tag_id = ?)",
			*req.TagID,
		)
	}

	var total int64
	query.Count(&total)

	var articles []model.DocArticle
	query.Order(req.OrderBy + " " + req.SortOrder).
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&articles)

	return articles, total
}

// FindBySlug returns a single article by slug.
func (r *ArticleRepository) FindBySlug(slug string) (*model.DocArticle, error) {
	var article model.DocArticle
	if err := r.db.Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// IncrementView bumps view by 1 (fire-and-forget).
func (r *ArticleRepository) IncrementView(id int) {
	r.db.Model(&model.DocArticle{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new article row (inside a tx).
func (r *ArticleRepository) Create(tx *gorm.DB, article *model.DocArticle) error {
	return tx.Create(article).Error
}

// UpdateFields updates arbitrary fields on an article row (inside a tx).
func (r *ArticleRepository) UpdateFields(tx *gorm.DB, id int, updates map[string]any) error {
	return tx.Model(&model.DocArticle{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteByID deletes an article row.
func (r *ArticleRepository) DeleteByID(id int) {
	r.db.Delete(&model.DocArticle{}, id)
}

// ReplaceTagRelations deletes existing tag relations and inserts the new set
// atomically inside the given tx.
func (r *ArticleRepository) ReplaceTagRelations(tx *gorm.DB, articleID int, tagIDs []int) {
	tx.Where("doc_article_id = ?", articleID).Delete(&model.DocArticleTagRelation{})
	for _, tagID := range tagIDs {
		tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.DocArticleTagRelation{
			DocArticleID: articleID, DocTagID: tagID,
		})
	}
}

// InsertTagRelations inserts tag relations for a newly created article.
func (r *ArticleRepository) InsertTagRelations(tx *gorm.DB, articleID int, tagIDs []int) {
	for _, tagID := range tagIDs {
		tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.DocArticleTagRelation{
			DocArticleID: articleID, DocTagID: tagID,
		})
	}
}

// DeleteTagRelationsByArticleID deletes all tag relations for an article.
func (r *ArticleRepository) DeleteTagRelationsByArticleID(articleID int) {
	r.db.Where("doc_article_id = ?", articleID).Delete(&model.DocArticleTagRelation{})
}
