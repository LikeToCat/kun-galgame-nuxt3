package handler

import (
	"net/url"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebsiteHandler struct {
	db *gorm.DB
}

func NewWebsiteHandler(db *gorm.DB) *WebsiteHandler {
	return &WebsiteHandler{db: db}
}

// ── Website CRUD ────────────────────────

// GetWebsites returns all websites as WebsiteCard[].
// GET /api/website
func (h *WebsiteHandler) GetWebsites(c *fiber.Ctx) error {
	type dbRow struct {
		ID          int
		Name        string
		URL         string
		Description string
		Icon        string
		AgeLimit    string
		CategoryID  int
	}
	var rows []dbRow
	h.db.Table("galgame_website").
		Select("id, name, url, description, icon, age_limit, category_id").
		Order("created DESC").
		Scan(&rows)

	// Batch load category names
	catIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		catIDs = append(catIDs, r.CategoryID)
	}
	var cats []struct {
		ID   int
		Name string
	}
	h.db.Table("galgame_website_category").
		Select("id, name").
		Where("id IN ?", catIDs).
		Scan(&cats)
	catMap := make(map[int]string, len(cats))
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	// Batch load tag level sums per website
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var tagSums []tagSum
	h.db.Table("galgame_website_tag_relation r").
		Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
		Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
		Group("r.galgame_website_id").
		Scan(&tagSums)
	levelMap := make(map[int]int, len(tagSums))
	for _, ts := range tagSums {
		levelMap[ts.WebsiteID] = ts.Total
	}

	type card struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Domain      string `json:"domain"`
		AgeLimit    string `json:"ageLimit"`
		Level       int    `json:"level"`
		Icon        string `json:"icon"`
		Price       int    `json:"price"`
		Category    string `json:"category"`
	}
	cards := make([]card, len(rows))
	for i, r := range rows {
		lvl := levelMap[r.ID]
		cards[i] = card{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Domain:      r.URL,
			AgeLimit:    r.AgeLimit,
			Level:       lvl,
			Icon:        r.Icon,
			Price:       lvl,
			Category:    catMap[r.CategoryID],
		}
	}
	return response.OK(c, cards)
}

// CreateWebsite creates a new website entry.
// POST /api/website
func (h *WebsiteHandler) CreateWebsite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Name        string `json:"name" validate:"required,max=233"`
		URL         string `json:"url" validate:"required,url,max=500"`
		Description string `json:"description" validate:"max=1000"`
		Icon        string `json:"icon" validate:"max=500"`
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
		Language    string `json:"language" validate:"max=10"`
		TagIDs      []int  `json:"tag_ids"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	parsedURL, _ := url.Parse(req.URL)
	domain := ""
	if parsedURL != nil {
		domain = parsedURL.Hostname()
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		website := model.GalgameWebsite{
			Name: req.Name, URL: req.URL, Description: req.Description,
			Icon: req.Icon, Language: req.Language, AgeLimit: req.AgeLimit,
			CategoryID: req.CategoryID, UserID: user.UID,
		}
		_ = domain // domain stored in jsonb field if needed
		if err := tx.Create(&website).Error; err != nil {
			return err
		}
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
				GalgameWebsiteID: website.ID, GalgameWebsiteTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("创建网站失败"))
	}

	return response.OKMessage(c, "网站创建成功")
}

// GetWebsiteDetail returns website detail by domain.
// GET /api/website/:domain
func (h *WebsiteHandler) GetWebsiteDetail(c *fiber.Ctx) error {
	domain := c.Params("domain")

	var website model.GalgameWebsite
	if err := h.db.Where("url ILIKE ?", "%"+domain+"%").First(&website).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该网站"))
	}

	go h.db.Model(&model.GalgameWebsite{}).Where("id = ?", website.ID).
		Update("view", gorm.Expr("view + 1"))

	// Category
	var category model.GalgameWebsiteCategory
	h.db.Where("id = ?", website.CategoryID).First(&category)

	// Tags
	var tagRels []model.GalgameWebsiteTagRelation
	h.db.Where("galgame_website_id = ?", website.ID).
		Preload("Tag").Find(&tagRels)
	tags := make([]fiber.Map, len(tagRels))
	for i, tr := range tagRels {
		tags[i] = fiber.Map{
			"id":          tr.Tag.ID,
			"name":        tr.Tag.Name,
			"description": tr.Tag.Description,
			"label":       tr.Tag.Label,
			"level":       tr.Tag.Level,
		}
	}

	// Comments with user
	type commentRow struct {
		ID         int    `gorm:"column:id"`
		Content    string `gorm:"column:content"`
		UserID     int    `gorm:"column:user_id"`
		UserName   string `gorm:"column:user_name"`
		UserAvatar string `gorm:"column:user_avatar"`
		Created    string `gorm:"column:created"`
		Updated    string `gorm:"column:updated"`
	}
	var comments []commentRow
	h.db.Table("galgame_website_comment c").
		Select(`c.id, c.content, c.user_id,
			u.name AS user_name, u.avatar AS user_avatar,
			c.created, c.updated`).
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`).
		Where("c.website_id = ?", website.ID).
		Order("c.created DESC").
		Scan(&comments)

	commentList := make([]fiber.Map, len(comments))
	for i, cm := range comments {
		commentList[i] = fiber.Map{
			"id":      cm.ID,
			"content": cm.Content,
			"user": fiber.Map{
				"id": cm.UserID, "name": cm.UserName, "avatar": cm.UserAvatar,
			},
			"created": cm.Created,
			"updated": cm.Updated,
		}
	}

	// Interactions
	userInfo := middleware.GetUser(c)
	isLiked, isFavorited := false, false
	if userInfo != nil {
		var lc, fc int64
		h.db.Model(&model.GalgameWebsiteLike{}).
			Where("user_id = ? AND website_id = ?", userInfo.UID, website.ID).Count(&lc)
		h.db.Model(&model.GalgameWebsiteFavorite{}).
			Where("user_id = ? AND website_id = ?", userInfo.UID, website.ID).Count(&fc)
		isLiked, isFavorited = lc > 0, fc > 0
	}

	return response.OK(c, fiber.Map{
		"id":          website.ID,
		"name":        website.Name,
		"url":         website.URL,
		"description": website.Description,
		"icon":        website.Icon,
		"view":        website.View,
		"language":    website.Language,
		"ageLimit":    website.AgeLimit,
		"category": fiber.Map{
			"id":          category.ID,
			"name":        category.Name,
			"label":       category.Label,
			"description": category.Description,
		},
		"tags":          tags,
		"likeCount":     website.LikeCount,
		"isLiked":       isLiked,
		"favoriteCount": website.FavoriteCount,
		"isFavorited":   isFavorited,
		"domain":        website.Domain,
		"createTime":    website.CreateTime,
		"comment":       commentList,
		"created":       website.CreatedAt,
		"updated":       website.UpdatedAt,
	})
}

// UpdateWebsite updates a website.
// PUT /api/website/:domain
func (h *WebsiteHandler) UpdateWebsite(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID   int    `json:"websiteId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required,max=233"`
		URL         string `json:"url" validate:"required,url,max=500"`
		Description string `json:"description" validate:"max=1000"`
		Icon        string `json:"icon" validate:"max=500"`
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
		Language    string `json:"language" validate:"max=10"`
		TagIDs      []int  `json:"tag_ids"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).Updates(map[string]any{
			"name": req.Name, "url": req.URL, "description": req.Description,
			"icon": req.Icon, "category_id": req.CategoryID,
			"age_limit": req.AgeLimit, "language": req.Language,
		})
		tx.Where("website_id = ?", req.WebsiteID).Delete(&model.GalgameWebsiteTagRelation{})
		for _, tagID := range req.TagIDs {
			tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
				GalgameWebsiteID: req.WebsiteID, GalgameWebsiteTagID: tagID,
			})
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, errors.ErrInternal("更新网站失败"))
	}

	return response.OKMessage(c, "网站更新成功")
}

// DeleteWebsite deletes a website.
// DELETE /api/website/:domain
func (h *WebsiteHandler) DeleteWebsite(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `query:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Delete(&model.GalgameWebsite{}, req.WebsiteID)
	return response.OKMessage(c, "网站已删除")
}

// ── Interactions ────────────────────────

// ToggleLike toggles website like.
// PUT /api/website/:domain/like
func (h *WebsiteHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `json:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameWebsiteLike
		result := tx.Where("user_id = ? AND website_id = ?", user.UID, req.WebsiteID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameWebsiteLike{UserID: user.UID, WebsiteID: req.WebsiteID})
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("like_count", gorm.Expr("like_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("like_count", gorm.Expr("like_count - 1"))
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite toggles website favorite.
// PUT /api/website/:domain/favorite
func (h *WebsiteHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		WebsiteID int `json:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.GalgameWebsiteFavorite
		result := tx.Where("user_id = ? AND website_id = ?", user.UID, req.WebsiteID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			tx.Create(&model.GalgameWebsiteFavorite{UserID: user.UID, WebsiteID: req.WebsiteID})
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("favorite_count", gorm.Expr("favorite_count + 1"))
		} else {
			tx.Delete(&existing)
			tx.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
				Update("favorite_count", gorm.Expr("favorite_count - 1"))
		}
		return nil
	})

	return response.OKMessage(c, "操作成功")
}

// GetComments returns nested comments for a website.
// GET /api/website/:domain/comment
func (h *WebsiteHandler) GetComments(c *fiber.Ctx) error {
	var req struct {
		WebsiteID int `query:"websiteId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	type commentRow struct {
		ID       int     `gorm:"column:id"`
		Content  string  `gorm:"column:content"`
		ParentID *int    `gorm:"column:parent_id"`
		UserID   int     `gorm:"column:user_id"`
		UserName string  `gorm:"column:user_name"`
		UserAvt  string  `gorm:"column:user_avatar"`
		Created  string  `gorm:"column:created"`
		Edited   *string `gorm:"column:edited"`
	}
	var rows []commentRow
	h.db.Table("galgame_website_comment c").
		Select(`c.id, c.content, c.parent_id, c.user_id,
			u.name AS user_name, u.avatar AS user_avatar,
			c.created, c.edited`).
		Joins(`LEFT JOIN "user" u ON u.id = c.user_id`).
		Where("c.website_id = ?", req.WebsiteID).
		Order("c.created DESC").
		Scan(&rows)

	// Build flat map then nest
	type comment struct {
		ID        int        `json:"id"`
		Content   string     `json:"content"`
		ParentID  *int       `json:"parentId"`
		UserID    int        `json:"userId"`
		WebsiteID int        `json:"websiteId"`
		Created   string     `json:"created"`
		Edited    *string    `json:"edited"`
		Reply     []*comment `json:"reply"`
		User      fiber.Map  `json:"user"`
		TargetUser any       `json:"targetUser"`
	}

	flat := make([]*comment, len(rows))
	idMap := map[int]*comment{}
	for i, r := range rows {
		cm := &comment{
			ID: r.ID, Content: r.Content, ParentID: r.ParentID,
			UserID: r.UserID, WebsiteID: req.WebsiteID,
			Created: r.Created, Edited: r.Edited,
			Reply: []*comment{},
			User: fiber.Map{"id": r.UserID, "name": r.UserName, "avatar": r.UserAvt},
			TargetUser: nil,
		}
		flat[i] = cm
		idMap[r.ID] = cm
	}

	var nested []*comment
	for _, cm := range flat {
		if cm.ParentID != nil {
			if parent, ok := idMap[*cm.ParentID]; ok {
				cm.TargetUser = parent.User
				parent.Reply = append(parent.Reply, cm)
				continue
			}
		}
		nested = append(nested, cm)
	}

	if nested == nil {
		nested = []*comment{}
	}

	return response.OK(c, nested)
}

// CreateComment creates a website comment.
// POST /api/website/:domain/comment
func (h *WebsiteHandler) CreateComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Content   string `json:"content" validate:"required,min=1,max=1007"`
		WebsiteID int    `json:"websiteId" validate:"required,min=1"`
		ParentID  *int   `json:"parentId"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	comment := model.GalgameWebsiteComment{
		Content: req.Content, WebsiteID: req.WebsiteID,
		UserID: user.UID, ParentID: req.ParentID,
	}
	if err := h.db.Create(&comment).Error; err != nil {
		return response.Error(c, errors.ErrInternal("发表评论失败"))
	}

	h.db.Model(&model.GalgameWebsite{}).Where("id = ?", req.WebsiteID).
		Update("comment_count", gorm.Expr("comment_count + 1"))

	return response.OK(c, comment)
}

// DeleteComment deletes a website comment.
// DELETE /api/website/:domain/comment
func (h *WebsiteHandler) DeleteComment(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `query:"commentId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var comment model.GalgameWebsiteComment
	if err := h.db.First(&comment, req.CommentID).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该评论"))
	}
	if comment.UserID != user.UID && user.Role < 2 {
		return response.Error(c, errors.ErrForbidden("您没有权限删除此评论"))
	}

	h.db.Delete(&comment)
	h.db.Model(&model.GalgameWebsite{}).Where("id = ?", comment.WebsiteID).
		Update("comment_count", gorm.Expr("comment_count - 1"))

	return response.OKMessage(c, "评论已删除")
}

// ── Category & Tag ──────────────────────

// GetWebsiteCategory returns a category with its websites.
// GET /api/website-category/:name
func (h *WebsiteHandler) GetWebsiteCategory(c *fiber.Ctx) error {
	name := c.Params("name")
	var category model.GalgameWebsiteCategory
	if err := h.db.Where("name = ?", name).First(&category).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该分类"))
	}

	type wsRow struct {
		ID          int    `gorm:"column:id"`
		Name        string `gorm:"column:name"`
		URL         string `gorm:"column:url"`
		Description string `gorm:"column:description"`
		Icon        string `gorm:"column:icon"`
		AgeLimit    string `gorm:"column:age_limit"`
	}
	var rows []wsRow
	h.db.Table("galgame_website").
		Select("id, name, url, description, icon, age_limit").
		Where("category_id = ?", category.ID).Scan(&rows)

	websiteIDs := make([]int, len(rows))
	for i, r := range rows {
		websiteIDs[i] = r.ID
	}

	// Tag level sums
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var tagSums []tagSum
	if len(websiteIDs) > 0 {
		h.db.Table("galgame_website_tag_relation r").
			Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
			Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
			Where("r.galgame_website_id IN ?", websiteIDs).
			Group("r.galgame_website_id").Scan(&tagSums)
	}
	levelMap := make(map[int]int, len(tagSums))
	for _, ts := range tagSums {
		levelMap[ts.WebsiteID] = ts.Total
	}

	cards := make([]fiber.Map, len(rows))
	for i, r := range rows {
		lvl := levelMap[r.ID]
		cards[i] = fiber.Map{
			"id": r.ID, "name": r.Name, "description": r.Description,
			"domain": r.URL, "ageLimit": r.AgeLimit,
			"level": lvl, "icon": r.Icon, "price": lvl,
			"category": category.Name,
		}
	}

	return response.OK(c, fiber.Map{
		"id":           category.ID,
		"name":         category.Name,
		"label":        category.Label,
		"description":  category.Description,
		"websiteCount": len(rows),
		"websites":     cards,
		"created":      category.CreatedAt,
		"updated":      category.UpdatedAt,
	})
}

// UpdateWebsiteCategory updates a website category.
// PUT /api/website-category
func (h *WebsiteHandler) UpdateWebsiteCategory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CategoryID  int    `json:"categoryId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required"`
		Label       string `json:"label"`
		Description string `json:"description"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Model(&model.GalgameWebsiteCategory{}).Where("id = ?", req.CategoryID).Updates(map[string]any{
		"name": req.Name, "label": req.Label, "description": req.Description,
	})
	return response.OKMessage(c, "分类更新成功")
}

// GetWebsiteTags returns website tags.
// GET /api/website-tag
func (h *WebsiteHandler) GetWebsiteTags(c *fiber.Ctx) error {
	var tags []model.GalgameWebsiteTag
	h.db.Order("id ASC").Find(&tags)
	return response.OK(c, tags)
}

// GetWebsiteTagDetail returns a tag with its websites.
// GET /api/website-tag/:name
func (h *WebsiteHandler) GetWebsiteTagDetail(c *fiber.Ctx) error {
	name := c.Params("name")
	var tag model.GalgameWebsiteTag
	if err := h.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return response.Error(c, errors.ErrNotFound("未找到该标签"))
	}

	// Find websites with this tag
	var rels []model.GalgameWebsiteTagRelation
	h.db.Where("galgame_website_tag_id = ?", tag.ID).Find(&rels)

	websiteIDs := make([]int, len(rels))
	for i, r := range rels {
		websiteIDs[i] = r.GalgameWebsiteID
	}

	type wsRow struct {
		ID          int    `gorm:"column:id"`
		Name        string `gorm:"column:name"`
		URL         string `gorm:"column:url"`
		Description string `gorm:"column:description"`
		Icon        string `gorm:"column:icon"`
		AgeLimit    string `gorm:"column:age_limit"`
		CategoryID  int    `gorm:"column:category_id"`
	}
	var websites []wsRow
	if len(websiteIDs) > 0 {
		h.db.Table("galgame_website").
			Select("id, name, url, description, icon, age_limit, category_id").
			Where("id IN ?", websiteIDs).Scan(&websites)
	}

	// Category names
	catIDs := make([]int, len(websites))
	for i, w := range websites {
		catIDs[i] = w.CategoryID
	}
	var cats []struct {
		ID   int
		Name string
	}
	if len(catIDs) > 0 {
		h.db.Table("galgame_website_category").
			Select("id, name").Where("id IN ?", catIDs).Scan(&cats)
	}
	catMap := make(map[int]string, len(cats))
	for _, ct := range cats {
		catMap[ct.ID] = ct.Name
	}

	// Tag level sums per website
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var tagSums []tagSum
	if len(websiteIDs) > 0 {
		h.db.Table("galgame_website_tag_relation r").
			Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
			Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
			Where("r.galgame_website_id IN ?", websiteIDs).
			Group("r.galgame_website_id").Scan(&tagSums)
	}
	levelMap := make(map[int]int, len(tagSums))
	for _, ts := range tagSums {
		levelMap[ts.WebsiteID] = ts.Total
	}

	cards := make([]fiber.Map, len(websites))
	for i, w := range websites {
		lvl := levelMap[w.ID]
		cards[i] = fiber.Map{
			"id": w.ID, "name": w.Name, "description": w.Description,
			"domain": w.URL, "ageLimit": w.AgeLimit,
			"level": lvl, "icon": w.Icon, "price": lvl,
			"category": catMap[w.CategoryID],
		}
	}

	return response.OK(c, fiber.Map{
		"id":           tag.ID,
		"name":         tag.Name,
		"label":        tag.Label,
		"level":        tag.Level,
		"description":  tag.Description,
		"websiteCount": len(websites),
		"websites":     cards,
		"created":      tag.CreatedAt,
		"updated":      tag.UpdatedAt,
	})
}

// CreateWebsiteTag creates a website tag.
// POST /api/website-tag
func (h *WebsiteHandler) CreateWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req model.GalgameWebsiteTag
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Create(&req)
	return response.OKMessage(c, "标签创建成功")
}

// UpdateWebsiteTag updates a website tag.
// PUT /api/website-tag
func (h *WebsiteHandler) UpdateWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		TagID       int    `json:"tagId" validate:"required,min=1"`
		Name        string `json:"name" validate:"required"`
		Label       string `json:"label"`
		Description string `json:"description"`
		Level       int    `json:"level"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	h.db.Model(&model.GalgameWebsiteTag{}).Where("id = ?", req.TagID).Updates(map[string]any{
		"name": req.Name, "label": req.Label, "description": req.Description, "level": req.Level,
	})
	return response.OKMessage(c, "标签更新成功")
}

// DeleteWebsiteTag deletes a website tag.
// DELETE /api/website-tag
func (h *WebsiteHandler) DeleteWebsiteTag(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		TagID int `query:"tagId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Where("tag_id = ?", req.TagID).Delete(&model.GalgameWebsiteTagRelation{})
	h.db.Delete(&model.GalgameWebsiteTag{}, req.TagID)
	return response.OKMessage(c, "标签已删除")
}
