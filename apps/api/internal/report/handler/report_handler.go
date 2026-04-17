package handler

import (
	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/report/dto"
	"kun-galgame-api/internal/report/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ReportHandler handles content report submission.
// No service layer — logic is just "validate + insert".
type ReportHandler struct {
	repo *repository.ReportRepository
}

func NewReportHandler(repo *repository.ReportRepository) *ReportHandler {
	return &ReportHandler{repo: repo}
}

// SubmitReport creates a content report.
// POST /api/report/submit
func (h *ReportHandler) SubmitReport(c *fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.SubmitReportRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	report := adminModel.Report{Reason: req.Reason, Type: req.Type}
	if err := h.repo.Create(&report); err != nil {
		return response.Error(c, errors.ErrInternal("提交举报失败"))
	}
	return response.OKMessage(c, "举报已提交")
}
