package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PollHandler struct {
	pollService *service.PollService
}

func NewPollHandler(pollService *service.PollService) *PollHandler {
	return &PollHandler{pollService: pollService}
}

// CreatePoll creates a new poll for a topic.
// POST /api/topic/:tid/poll
func (h *PollHandler) CreatePoll(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreatePollRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.pollService.CreatePoll(c.Context(), user.UID, user.Role, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "投票创建成功")
}

// GetPollsByTopic returns all polls for a topic.
// GET /api/topic/:tid/poll/topic
func (h *PollHandler) GetPollsByTopic(c *fiber.Ctx) error {
	var req dto.GetPollByTopicRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	userInfo := middleware.GetUser(c)

	polls, appErr := h.pollService.GetPollsByTopic(c.Context(), req.TopicID, userInfo)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, polls)
}

// Vote submits a vote on a poll.
// POST /api/topic/:tid/poll/vote
func (h *PollHandler) Vote(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	_ = user

	var req dto.VoteRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.pollService.Vote(c.Context(), user.UID, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "投票成功")
}

// DeletePoll deletes a poll and all its votes.
// DELETE /api/topic/:tid/poll
func (h *PollHandler) DeletePoll(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		PollID int `query:"poll_id" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.pollService.DeletePoll(c.Context(), user.UID, user.Role, req.PollID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "投票已删除")
}

// GetVoteLog returns paginated vote log for a poll.
// GET /api/topic/:tid/poll/log
func (h *PollHandler) GetVoteLog(c *fiber.Ctx) error {
	var req dto.GetPollLogRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	userInfo := middleware.GetUser(c)

	entries, total, appErr := h.pollService.GetVoteLog(c.Context(), req.PollID, req.Page, req.Limit, userInfo)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.Paginated(c, entries, total)
}
