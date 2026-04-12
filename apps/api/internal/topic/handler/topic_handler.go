package handler

import (
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type TopicHandler struct {
	topicService *service.TopicService
}

func NewTopicHandler(topicService *service.TopicService) *TopicHandler {
	return &TopicHandler{topicService: topicService}
}

// GetList returns paginated topic list.
// GET /api/topic
func (h *TopicHandler) GetList(c *fiber.Ctx) error {
	var req dto.ListTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	// TODO: read NSFW cookie
	isNSFW := false

	items, total, appErr := h.topicService.GetList(c.Context(), &req, isNSFW)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.Paginated(c, items, total)
}

// GetDetail returns a single topic with all associated data.
// GET /api/topic/:tid
func (h *TopicHandler) GetDetail(c *fiber.Ctx) error {
	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的话题 ID"))
	}

	userInfo := middleware.GetUser(c)

	detail, appErr := h.topicService.GetDetail(c.Context(), tid, userInfo)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, detail)
}

// Create creates a new topic.
// POST /api/topic
func (h *TopicHandler) Create(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateTopicRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	topicID, appErr := h.topicService.Create(c.Context(), user.UID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, topicID)
}

// Update edits an existing topic.
// PUT /api/topic/:tid
func (h *TopicHandler) Update(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无��的话题 ID"))
	}

	var req dto.UpdateTopicRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.topicService.Update(c.Context(), user.UID, user.Role, tid, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "话题更新成功")
}

// ToggleLike toggles like on a topic.
// PUT /api/topic/:tid/like
func (h *TopicHandler) ToggleLike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的话题 ID"))
	}

	if appErr := h.topicService.ToggleLike(c.Context(), user.UID, tid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}

// ToggleDislike toggles dislike on a topic.
// PUT /api/topic/:tid/dislike
func (h *TopicHandler) ToggleDislike(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无��的话题 ID"))
	}

	if appErr := h.topicService.ToggleDislike(c.Context(), user.UID, tid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "��作成功")
}

// Upvote pushes a topic (costs sender moemoepoints).
// PUT /api/topic/:tid/upvote
func (h *TopicHandler) Upvote(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效��话�� ID"))
	}

	if appErr := h.topicService.Upvote(c.Context(), user.UID, tid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "推话题成功")
}

// ToggleFavorite toggles favorite on a topic.
// PUT /api/topic/:tid/favorite
func (h *TopicHandler) ToggleFavorite(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的话题 ID"))
	}

	if appErr := h.topicService.ToggleFavorite(c.Context(), user.UID, tid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}

// ToggleHide hides or unhides a topic.
// PUT /api/topic/:tid/hide
func (h *TopicHandler) ToggleHide(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("���效的话题 ID"))
	}

	if appErr := h.topicService.ToggleHide(c.Context(), user.UID, user.Role, tid); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}

// SetBestAnswer marks a reply as the best answer.
// PUT /api/topic/:tid/best-answer
func (h *TopicHandler) SetBestAnswer(c *fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	tid, err := strconv.Atoi(c.Params("tid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的话�� ID"))
	}

	var req dto.BestAnswerRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.topicService.SetBestAnswer(c.Context(), user.UID, tid, req.ReplyID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "已设置最佳回答")
}
