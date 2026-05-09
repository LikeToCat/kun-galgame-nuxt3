package handler

import (
	"kun-galgame-api/internal/section/dto"
	"kun-galgame-api/internal/section/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type SectionHandler struct {
	sectionService *service.SectionService
}

func NewSectionHandler(sectionService *service.SectionService) *SectionHandler {
	return &SectionHandler{sectionService: sectionService}
}

// GetSectionTopics returns topics filtered by section.
// GET /api/section
func (h *SectionHandler) GetSectionTopics(c *fiber.Ctx) error {
	var req dto.SectionTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resp := h.sectionService.GetSectionTopics(c.Context(), &req)
	return response.OK(c, fiber.Map{
		"topics": resp.Topics,
		"total":  resp.Total,
	})
}

// GetCategories returns topic category stats.
// GET /api/category
func (h *SectionHandler) GetCategories(c *fiber.Ctx) error {
	var req dto.CategoriesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.sectionService.GetCategoryStats(req.Category))
}
