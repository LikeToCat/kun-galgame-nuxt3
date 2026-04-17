package handler

import (
	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/update/dto"
	"kun-galgame-api/internal/update/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UpdateHandler handles update log + todo list routes.
// No service layer — these endpoints are pure admin CRUD with
// no business logic beyond straight DB ops.
type UpdateHandler struct {
	repo *repository.UpdateRepository
}

func NewUpdateHandler(repo *repository.UpdateRepository) *UpdateHandler {
	return &UpdateHandler{repo: repo}
}

// ── History ─────────────────────────────

// GetHistory returns paginated update logs.
// GET /api/update/history
func (h *UpdateHandler) GetHistory(c *fiber.Ctx) error {
	var req dto.ListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	logs := h.repo.FindHistoryPaginated(req.Page, req.Limit)
	total := h.repo.CountHistory()

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

	var req dto.CreateHistoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	log := adminModel.UpdateLog{
		Type: req.Type, Version: req.Version,
		ContentEnUS: req.ContentEnUS, ContentJaJP: req.ContentJaJP,
		ContentZhCN: req.ContentZhCN, ContentZhTW: req.ContentZhTW,
		UserID: user.UID,
	}
	if err := h.repo.CreateHistory(&log); err != nil {
		return response.Error(c, errors.ErrInternal("创建更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已创建")
}

// DeleteHistory deletes an update log.
// DELETE /api/update/history
func (h *UpdateHandler) DeleteHistory(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteHistoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.repo.DeleteHistory(req.ID)
	return response.OKMessage(c, "更新日志已删除")
}

// ── Todo ────────────────────────────────

// GetTodos returns paginated todo list.
// GET /api/update/todo
func (h *UpdateHandler) GetTodos(c *fiber.Ctx) error {
	var req dto.ListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todos := h.repo.FindTodosPaginated(req.Page, req.Limit)
	total := h.repo.CountTodos()

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

	var req dto.CreateTodoRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo := adminModel.Todo{
		Type: req.Type, Status: req.Status,
		ContentEnUS: req.ContentEnUS, ContentJaJP: req.ContentJaJP,
		ContentZhCN: req.ContentZhCN, ContentZhTW: req.ContentZhTW,
		UserID: user.UID,
	}
	if err := h.repo.CreateTodo(&todo); err != nil {
		return response.Error(c, errors.ErrInternal("创建待办失败"))
	}
	return response.OKMessage(c, "待办已创建")
}

// DeleteTodo deletes a todo item.
// DELETE /api/update/todo
func (h *UpdateHandler) DeleteTodo(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteTodoRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	h.repo.DeleteTodo(req.ID)
	return response.OKMessage(c, "待办已删除")
}
