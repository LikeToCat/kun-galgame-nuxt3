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

type MiscHandler struct {
	db *gorm.DB
}

func NewMiscHandler(db *gorm.DB) *MiscHandler {
	return &MiscHandler{db: db}
}

// SubmitReport creates a content report.
// POST /api/report/submit
func (h *MiscHandler) SubmitReport(c *fiber.Ctx) error {
	_, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		Reason string `json:"reason" validate:"required,max=1000"`
		Type   string `json:"type" validate:"required,max=100"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	report := adminModel.Report{Reason: req.Reason, Type: req.Type}
	if err := h.db.Create(&report).Error; err != nil {
		return response.Error(c, errors.ErrInternal("提交举报失败"))
	}
	return response.OKMessage(c, "举报已提交")
}

// GetTopicRSS returns recent topics for RSS feed.
// GET /api/rss/topic
func (h *MiscHandler) GetTopicRSS(c *fiber.Ctx) error {
	type rssTopic struct {
		ID          int       `json:"id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		UserID      int       `json:"userId"`
		UserName    string    `json:"userName"`
		Created     time.Time `json:"created"`
	}

	var topics []rssTopic
	h.db.Table("topic t").
		Select(`t.id, t.title, SUBSTRING(t.content, 1, 233) AS description,
			t.user_id, u.name AS user_name, t.created`).
		Joins(`LEFT JOIN "user" u ON u.id = t.user_id`).
		Where("t.status != 1 AND t.is_nsfw = false").
		Order("t.created DESC").
		Limit(10).
		Find(&topics)

	return response.OK(c, topics)
}
