package common

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UpdateHandler struct {
	db *gorm.DB
}

func NewUpdateHandler(db *gorm.DB) *UpdateHandler {
	return &UpdateHandler{db: db}
}

// ── History ─────────────────────────────

// GetHistory returns paginated update logs.
// GET /api/update/history
func (h *UpdateHandler) GetHistory(c *fiber.Ctx) error {
	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var logs []adminModel.UpdateLog
	var total int64
	h.db.Model(&adminModel.UpdateLog{}).Count(&total)
	h.db.Order("created DESC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&logs)

	return response.OK(c, fiber.Map{
		"updates": logs,
		"total":   total,
	})
}

// CreateHistory creates an update log.
// POST /api/update/history
func (h *UpdateHandler) CreateHistory(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Type        string `json:"type" validate:"required"`
		Version     string `json:"version"`
		ContentEnUS string `json:"content_en_us"`
		ContentJaJP string `json:"content_ja_jp"`
		ContentZhCN string `json:"content_zh_cn"`
		ContentZhTW string `json:"content_zh_tw"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	log := adminModel.UpdateLog{
		Type: req.Type, Version: req.Version,
		ContentEnUS: req.ContentEnUS, ContentJaJP: req.ContentJaJP,
		ContentZhCN: req.ContentZhCN, ContentZhTW: req.ContentZhTW,
		UserID: user.UID,
	}
	if err := h.db.Create(&log).Error; err != nil {
		return response.Error(c, errors.ErrInternal("创建更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已创建")
}

// DeleteHistory deletes an update log.
// DELETE /api/update/history
func (h *UpdateHandler) DeleteHistory(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ID int `query:"updateLogId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Delete(&adminModel.UpdateLog{}, req.ID)
	return response.OKMessage(c, "更新日志已删除")
}

// ── Todo ────────────────────────────────

// GetTodos returns paginated todo list.
// GET /api/update/todo
func (h *UpdateHandler) GetTodos(c *fiber.Ctx) error {
	var req struct {
		Page  int `query:"page" validate:"min=1"`
		Limit int `query:"limit" validate:"min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	var todos []adminModel.Todo
	var total int64
	h.db.Model(&adminModel.Todo{}).Count(&total)
	h.db.Order("created DESC").
		Offset((req.Page - 1) * req.Limit).Limit(req.Limit).
		Find(&todos)

	return response.OK(c, fiber.Map{
		"todos": todos,
		"total": total,
	})
}

// CreateTodo creates a todo item.
// POST /api/update/todo
func (h *UpdateHandler) CreateTodo(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Type        string `json:"type" validate:"required"`
		Status      int    `json:"status"`
		ContentEnUS string `json:"content_en_us"`
		ContentJaJP string `json:"content_ja_jp"`
		ContentZhCN string `json:"content_zh_cn"`
		ContentZhTW string `json:"content_zh_tw"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo := adminModel.Todo{
		Type: req.Type, Status: req.Status,
		ContentEnUS: req.ContentEnUS, ContentJaJP: req.ContentJaJP,
		ContentZhCN: req.ContentZhCN, ContentZhTW: req.ContentZhTW,
		UserID: user.UID,
	}
	if req.Status == 2 {
		now := time.Now()
		todo.CompletedTime = &now
	}
	if err := h.db.Create(&todo).Error; err != nil {
		return response.Error(c, errors.ErrInternal("创建待办失败"))
	}
	return response.OKMessage(c, "待办已创建")
}

// DeleteTodo deletes a todo item.
// DELETE /api/update/todo
func (h *UpdateHandler) DeleteTodo(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		ID int `query:"todoId" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.db.Delete(&adminModel.Todo{}, req.ID)
	return response.OKMessage(c, "待办已删除")
}
