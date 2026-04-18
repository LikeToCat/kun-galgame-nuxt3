package handler

import (
	"kun-galgame-api/internal/unmoe/dto"
	"kun-galgame-api/internal/unmoe/repository"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UnmoeHandler handles the public unmoe translator log endpoint.
// No service layer — query + projection only.
type UnmoeHandler struct {
	repo *repository.UnmoeRepository
}

func NewUnmoeHandler(repo *repository.UnmoeRepository) *UnmoeHandler {
	return &UnmoeHandler{repo: repo}
}

// GetLogs returns paginated translator logs.
// GET /api/unmoe
func (h *UnmoeHandler) GetLogs(c *fiber.Ctx) error {
	var req dto.ListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	rows, total := h.repo.FindPaginated(req.Page, req.Limit)
	logs := make([]dto.UnmoeLog, len(rows))
	for i, r := range rows {
		logs[i] = dto.UnmoeLog{
			ID: r.ID, Name: r.Name, Result: r.Result, Created: r.Created,
			User: dto.UnmoeUser{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Description: dto.UnmoeDescription{
				EnUs: r.DescEnUs, JaJp: r.DescJaJp,
				ZhCn: r.DescZhCn, ZhTw: r.DescZhTw,
			},
		}
	}

	return response.OK(c, dto.ListResponse{Logs: logs, Total: total})
}
