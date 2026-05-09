package handler

import (
	"kun-galgame-api/internal/unmoe/dto"
	"kun-galgame-api/internal/unmoe/repository"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UnmoeHandler handles the public unmoe translator log endpoint.
// No service layer — query + projection only.
type UnmoeHandler struct {
	repo       *repository.UnmoeRepository
	userClient *userclient.Client
}

func NewUnmoeHandler(repo *repository.UnmoeRepository, userClient *userclient.Client) *UnmoeHandler {
	return &UnmoeHandler{repo: repo, userClient: userClient}
}

// GetLogs returns paginated translator logs. Identity is hydrated from OAuth
// via userclient since the repo no longer joins on the user table.
// GET /api/unmoe
func (h *UnmoeHandler) GetLogs(c *fiber.Ctx) error {
	var req dto.ListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	rows, total := h.repo.FindPaginated(req.Page, req.Limit)
	uids := userclient.CollectIDs(rows, func(r repository.UnmoeRow) int { return r.UserID })
	userMap := h.userClient.Hydrate(c.Context(), uids)

	logs := make([]dto.UnmoeLog, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		logs = append(logs, dto.UnmoeLog{
			ID: r.ID, Name: r.Name, Result: r.Result, Created: r.Created,
			User: dto.UnmoeUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Description: dto.UnmoeDescription{
				EnUs: r.DescEnUs, JaJp: r.DescJaJp,
				ZhCn: r.DescZhCn, ZhTw: r.DescZhTw,
			},
		})
	}

	return response.OK(c, dto.ListResponse{Logs: logs, Total: total})
}
