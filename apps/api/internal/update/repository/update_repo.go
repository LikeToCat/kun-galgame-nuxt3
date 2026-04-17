package repository

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type UpdateRepository struct {
	db *gorm.DB
}

func NewUpdateRepository(db *gorm.DB) *UpdateRepository {
	return &UpdateRepository{db: db}
}

// ── History ─────────────────────────────

// CountHistory returns total update log count.
func (r *UpdateRepository) CountHistory() int64 {
	var total int64
	r.db.Model(&adminModel.UpdateLog{}).Count(&total)
	return total
}

// FindHistoryPaginated returns paginated update logs ordered by created DESC.
func (r *UpdateRepository) FindHistoryPaginated(page, limit int) []adminModel.UpdateLog {
	var logs []adminModel.UpdateLog
	r.db.Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&logs)
	return logs
}

// CreateHistory inserts a new update log.
func (r *UpdateRepository) CreateHistory(log *adminModel.UpdateLog) error {
	return r.db.Create(log).Error
}

// DeleteHistory deletes an update log by ID.
func (r *UpdateRepository) DeleteHistory(id int) {
	r.db.Delete(&adminModel.UpdateLog{}, id)
}

// ── Todo ────────────────────────────────

// CountTodos returns total todo count.
func (r *UpdateRepository) CountTodos() int64 {
	var total int64
	r.db.Model(&adminModel.Todo{}).Count(&total)
	return total
}

// FindTodosPaginated returns paginated todos ordered by created DESC.
func (r *UpdateRepository) FindTodosPaginated(page, limit int) []adminModel.Todo {
	var todos []adminModel.Todo
	r.db.Order("created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&todos)
	return todos
}

// CreateTodo inserts a new todo. If status is 2 (completed), sets CompletedTime.
func (r *UpdateRepository) CreateTodo(todo *adminModel.Todo) error {
	if todo.Status == 2 {
		now := time.Now()
		todo.CompletedTime = &now
	}
	return r.db.Create(todo).Error
}

// DeleteTodo deletes a todo by ID.
func (r *UpdateRepository) DeleteTodo(id int) {
	r.db.Delete(&adminModel.Todo{}, id)
}
