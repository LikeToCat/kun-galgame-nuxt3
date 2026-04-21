package repository

import (
	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
)

type WebsiteRepository struct {
	db *gorm.DB
}

func NewWebsiteRepository(db *gorm.DB) *WebsiteRepository {
	return &WebsiteRepository{db: db}
}

func (r *WebsiteRepository) DB() *gorm.DB { return r.db }

// GetURL returns the website's URL slug (used as the link key in the
// frontend `/website/:url` route). Empty on miss.
func (r *WebsiteRepository) GetURL(id int) string {
	var url string
	r.db.Model(&model.GalgameWebsite{}).Where("id = ?", id).
		Select("url").Scan(&url)
	return url
}

// ──────────────────────────────────────────
// Row projections
// ──────────────────────────────────────────

// WebsiteListRow is the slim projection used by list queries.
type WebsiteListRow struct {
	ID          int    `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	URL         string `gorm:"column:url"`
	Description string `gorm:"column:description"`
	Icon        string `gorm:"column:icon"`
	AgeLimit    string `gorm:"column:age_limit"`
	CategoryID  int    `gorm:"column:category_id"`
}

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// FindAll returns all websites as the slim list projection, ordered by created DESC.
func (r *WebsiteRepository) FindAll() []WebsiteListRow {
	var rows []WebsiteListRow
	r.db.Table("galgame_website").
		Select("id, name, url, description, icon, age_limit, category_id").
		Order("created DESC").
		Scan(&rows)
	return rows
}

// FindByCategoryID returns websites matching a category ID.
func (r *WebsiteRepository) FindByCategoryID(categoryID int) []WebsiteListRow {
	var rows []WebsiteListRow
	r.db.Table("galgame_website").
		Select("id, name, url, description, icon, age_limit, category_id").
		Where("category_id = ?", categoryID).
		Scan(&rows)
	return rows
}

// FindByIDs returns websites matching a list of IDs.
func (r *WebsiteRepository) FindByIDs(ids []int) []WebsiteListRow {
	if len(ids) == 0 {
		return nil
	}
	var rows []WebsiteListRow
	r.db.Table("galgame_website").
		Select("id, name, url, description, icon, age_limit, category_id").
		Where("id IN ?", ids).
		Scan(&rows)
	return rows
}

// FindByDomain returns a single website whose URL contains the given substring.
func (r *WebsiteRepository) FindByDomain(domain string) (*model.GalgameWebsite, error) {
	var website model.GalgameWebsite
	if err := r.db.Where("url ILIKE ?", "%"+domain+"%").First(&website).Error; err != nil {
		return nil, err
	}
	return &website, nil
}

// IncrementView bumps view by 1 (no-tx, fire-and-forget).
func (r *WebsiteRepository) IncrementView(id int) {
	r.db.Model(&model.GalgameWebsite{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new website row (inside a tx).
func (r *WebsiteRepository) Create(tx *gorm.DB, website *model.GalgameWebsite) error {
	return tx.Create(website).Error
}

// UpdateFields updates arbitrary fields on a website row (inside a tx).
func (r *WebsiteRepository) UpdateFields(tx *gorm.DB, id int, updates map[string]any) {
	tx.Model(&model.GalgameWebsite{}).Where("id = ?", id).Updates(updates)
}

// DeleteByID deletes a website row.
func (r *WebsiteRepository) DeleteByID(id int) {
	r.db.Delete(&model.GalgameWebsite{}, id)
}

// AdjustLikeCount bumps like_count by delta (inside a tx).
func (r *WebsiteRepository) AdjustLikeCount(tx *gorm.DB, id, delta int) {
	tx.Model(&model.GalgameWebsite{}).Where("id = ?", id).
		Update("like_count", gorm.Expr("like_count + ?", delta))
}

// AdjustFavoriteCount bumps favorite_count by delta (inside a tx).
func (r *WebsiteRepository) AdjustFavoriteCount(tx *gorm.DB, id, delta int) {
	tx.Model(&model.GalgameWebsite{}).Where("id = ?", id).
		Update("favorite_count", gorm.Expr("favorite_count + ?", delta))
}

// AdjustCommentCount bumps comment_count by delta.
func (r *WebsiteRepository) AdjustCommentCount(id, delta int) {
	r.db.Model(&model.GalgameWebsite{}).Where("id = ?", id).
		Update("comment_count", gorm.Expr("comment_count + ?", delta))
}

// ──────────────────────────────────────────
// Interactions
// ──────────────────────────────────────────

// FindLike returns a like row for (userID, websiteID).
func (r *WebsiteRepository) FindLike(tx *gorm.DB, userID, websiteID int) (*model.GalgameWebsiteLike, error) {
	var like model.GalgameWebsiteLike
	if err := tx.Where("user_id = ? AND website_id = ?", userID, websiteID).First(&like).Error; err != nil {
		return nil, err
	}
	return &like, nil
}

// FindFavorite returns a favorite row for (userID, websiteID).
func (r *WebsiteRepository) FindFavorite(tx *gorm.DB, userID, websiteID int) (*model.GalgameWebsiteFavorite, error) {
	var fav model.GalgameWebsiteFavorite
	if err := tx.Where("user_id = ? AND website_id = ?", userID, websiteID).First(&fav).Error; err != nil {
		return nil, err
	}
	return &fav, nil
}

// CreateLike inserts a like row (inside a tx).
func (r *WebsiteRepository) CreateLike(tx *gorm.DB, userID, websiteID int) {
	tx.Create(&model.GalgameWebsiteLike{UserID: userID, WebsiteID: websiteID})
}

// DeleteLike deletes an existing like row (inside a tx).
func (r *WebsiteRepository) DeleteLike(tx *gorm.DB, like *model.GalgameWebsiteLike) {
	tx.Delete(like)
}

// CreateFavorite inserts a favorite row (inside a tx).
func (r *WebsiteRepository) CreateFavorite(tx *gorm.DB, userID, websiteID int) {
	tx.Create(&model.GalgameWebsiteFavorite{UserID: userID, WebsiteID: websiteID})
}

// DeleteFavorite deletes an existing favorite row (inside a tx).
func (r *WebsiteRepository) DeleteFavorite(tx *gorm.DB, fav *model.GalgameWebsiteFavorite) {
	tx.Delete(fav)
}

// HasLike returns whether a like row exists.
func (r *WebsiteRepository) HasLike(userID, websiteID int) bool {
	var c int64
	r.db.Model(&model.GalgameWebsiteLike{}).
		Where("user_id = ? AND website_id = ?", userID, websiteID).Count(&c)
	return c > 0
}

// HasFavorite returns whether a favorite row exists.
func (r *WebsiteRepository) HasFavorite(userID, websiteID int) bool {
	var c int64
	r.db.Model(&model.GalgameWebsiteFavorite{}).
		Where("user_id = ? AND website_id = ?", userID, websiteID).Count(&c)
	return c > 0
}
