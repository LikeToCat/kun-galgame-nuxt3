package handler

import (
	"kun-galgame-api/internal/rss/repository"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RSSHandler handles RSS feed routes.
// No service layer — logic is a single query with fixed filters.
type RSSHandler struct {
	repo *repository.RSSRepository
}

func NewRSSHandler(repo *repository.RSSRepository) *RSSHandler {
	return &RSSHandler{repo: repo}
}

// GetTopicRSS returns recent topics for RSS feed.
// GET /api/rss/topic
func (h *RSSHandler) GetTopicRSS(c *fiber.Ctx) error {
	return response.OK(c, h.repo.FindRecentSFWTopics())
}
