package service

import (
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
)

// ──────────────────────────────────────────
// Poll mappers
// ──────────────────────────────────────────

// buildPollResponse assembles a TopicPollResponse from a poll model and the
// associated option/voter data loaded via the repository. It does not perform
// any DB writes; callers pass in the logged-in user context via uid/role.
func (s *PollService) buildPollResponse(poll *topicModel.TopicPoll, uid, role int) dto.TopicPollResponse {
	options, _ := s.pollRepo.FindOptionsByPollID(poll.ID)
	hasVoted, _ := s.pollRepo.HasUserVoted(poll.ID, uid)
	canView := canViewResults(poll, uid, role, hasVoted)

	var userVotedOptionIDs map[int]bool
	if uid > 0 {
		votedIDs, _ := s.pollRepo.FindUserVoteOptionIDs(poll.ID, uid)
		userVotedOptionIDs = make(map[int]bool, len(votedIDs))
		for _, id := range votedIDs {
			userVotedOptionIDs[id] = true
		}
	}

	optionResponses := make([]dto.PollOptionResponse, len(options))
	for i, opt := range options {
		var voteCount *int
		if canView {
			vc := opt.VoteCount
			voteCount = &vc
		}
		optionResponses[i] = dto.PollOptionResponse{
			ID:        opt.ID,
			Text:      opt.Text,
			VoteCount: voteCount,
			IsVoted:   userVotedOptionIDs[opt.ID],
		}
	}

	var voters []dto.KunUser
	var votersCount int
	var totalVoteCount *int
	if canView {
		if !poll.IsAnonymous {
			voters, _ = s.pollRepo.FindDistinctVoters(poll.ID, 5)
		}
		vc, _ := s.pollRepo.CountDistinctVoters(poll.ID)
		votersCount = vc
		tc, _ := s.pollRepo.CountTotalVotes(poll.ID)
		totalVoteCount = &tc
	}
	if voters == nil {
		voters = []dto.KunUser{}
	}

	creator, _ := s.pollRepo.FindUserBrief(poll.UserID)

	return dto.TopicPollResponse{
		ID: poll.ID, Title: poll.Title, Description: poll.Description,
		MinChoice: poll.MinChoice, MaxChoice: poll.MaxChoice,
		Deadline: poll.Deadline, Type: poll.Type, Status: poll.Status,
		ResultVisibility: poll.ResultVisibility,
		IsAnonymous:      poll.IsAnonymous, CanChangeVote: poll.CanChangeVote,
		TopicID: poll.TopicID, Created: poll.CreatedAt, Updated: poll.UpdatedAt,
		User: creator, Options: optionResponses,
		HasVoted: hasVoted, Voters: voters,
		VotersCount: votersCount, VoteCount: totalVoteCount,
	}
}

// canViewResults returns true if the caller is allowed to see vote counts /
// voter identities according to the poll's result_visibility setting.
func canViewResults(poll *topicModel.TopicPoll, uid, role int, hasVoted bool) bool {
	if uid == poll.UserID || role > 1 {
		return true
	}
	isPollFinished := poll.Status == "closed" ||
		(poll.Deadline != nil && time.Now().After(*poll.Deadline))

	switch poll.ResultVisibility {
	case "always":
		return true
	case "after_vote":
		return hasVoted
	case "after_deadline":
		return isPollFinished
	default:
		return false
	}
}

// ──────────────────────────────────────────
// Reply mappers
// ──────────────────────────────────────────

// buildReplyResponses turns a batch of ReplyRow into TopicReplyResponse DTOs.
// Fetches targets/comments/like-status via the repository in bulk.
func (s *ReplyService) buildReplyResponses(
	rows []repository.ReplyRow,
	topic *topicModel.Topic,
	userInfo *middleware.UserInfo,
) []dto.TopicReplyResponse {
	if len(rows) == 0 {
		return nil
	}

	replyIDs := make([]int, len(rows))
	for i, r := range rows {
		replyIDs[i] = r.ID
	}

	targetMap, _ := s.replyRepo.FindTargetsByReplyIDs(replyIDs)
	commentMap, _ := s.commentRepo.FindCommentsByReplyIDs(replyIDs)

	var likeMap, dislikeMap map[int]bool
	var commentLikeMap map[int]bool
	if userInfo != nil {
		likeMap, _ = s.replyRepo.FindReplyLikeStatus(userInfo.UID, replyIDs)
		dislikeMap, _ = s.replyRepo.FindReplyDislikeStatus(userInfo.UID, replyIDs)

		var commentIDs []int
		for _, comments := range commentMap {
			for _, c := range comments {
				commentIDs = append(commentIDs, c.ID)
			}
		}
		commentLikeMap, _ = s.commentRepo.FindCommentLikeStatus(userInfo.UID, commentIDs)
	}

	responses := make([]dto.TopicReplyResponse, len(rows))
	for i, r := range rows {
		var targets []dto.ReplyTargetResponse
		if ts, ok := targetMap[r.ID]; ok {
			for _, t := range ts {
				preview := truncate(t.TargetContent, 150)
				targets = append(targets, dto.ReplyTargetResponse{
					ID:                   t.TargetReplyID,
					Floor:                t.TargetFloor,
					User:                 dto.KunUser{ID: t.TargetUserID, Name: t.TargetUserName, Avatar: t.TargetUserAvatar},
					ContentPreview:       preview,
					ReplyContentMarkdown: t.Content,
					ReplyContentHtml:     markdown.Render(t.Content),
				})
			}
		}
		if targets == nil {
			targets = []dto.ReplyTargetResponse{}
		}

		var comments []dto.TopicCommentResponse
		if cs, ok := commentMap[r.ID]; ok {
			for _, c := range cs {
				isLiked := false
				if commentLikeMap != nil {
					isLiked = commentLikeMap[c.ID]
				}
				comments = append(comments, dto.TopicCommentResponse{
					ID:         c.ID,
					ReplyID:    c.TopicReplyID,
					TopicID:    c.TopicID,
					User:       dto.KunUser{ID: c.UserID, Name: c.UserName, Avatar: c.UserAvatar},
					TargetUser: dto.KunUser{ID: c.TargetUserID, Name: c.TargetUserName, Avatar: c.TargetAvatar},
					Content:    c.Content,
					IsLiked:    isLiked,
					LikeCount:  c.LikeCount,
					Created:    c.CreatedAt,
				})
			}
		}
		if comments == nil {
			comments = []dto.TopicCommentResponse{}
		}

		isPinned := topic != nil && topic.PinnedReplyID != nil && *topic.PinnedReplyID == r.ID
		isBestAnswer := topic != nil && topic.BestAnswerID != nil && *topic.BestAnswerID == r.ID

		responses[i] = dto.TopicReplyResponse{
			ID:      r.ID,
			TopicID: r.TopicID,
			Floor:   r.Floor,
			User: dto.KunUserWithMoemoepoint{
				ID: r.UserID, Name: r.UserName,
				Avatar: r.UserAvatar, Moemoepoint: r.UserMoemoepoint,
			},
			Edited:          r.Edited,
			ContentMarkdown: r.Content,
			ContentHtml:     markdown.Render(r.Content),
			LikeCount:       r.LikeCount,
			IsLiked:         likeMap[r.ID],
			DislikeCount:    r.DislikeCount,
			IsDisliked:      dislikeMap[r.ID],
			Comments:        comments,
			Targets:         targets,
			IsPinned:        isPinned,
			IsBestAnswer:    isBestAnswer,
			Created:         r.CreatedAt,
		}
	}
	return responses
}

// ──────────────────────────────────────────
// Topic mappers
// ──────────────────────────────────────────

// toTopicCard maps a TopicCardRow with its tag/section slices to a TopicCard DTO.
// Shared by GetList and GetResourceList.
func toTopicCard(r repository.TopicCardRow, tags, sections []string, isPollTopic bool) dto.TopicCard {
	if tags == nil {
		tags = []string{}
	}
	if sections == nil {
		sections = []string{}
	}
	return dto.TopicCard{
		ID:       r.ID,
		Title:    r.Title,
		View:     r.View,
		Tags:     tags,
		Sections: sections,
		User: dto.KunUser{
			ID:     r.UserID,
			Name:   r.UserName,
			Avatar: r.UserAvatar,
		},
		Status:           r.Status,
		HasBestAnswer:    r.BestAnswerID != nil,
		IsPollTopic:      isPollTopic,
		IsNSFW:           r.IsNSFW,
		LikeCount:        r.LikeCount,
		ReplyCount:       r.ReplyCount,
		CommentCount:     r.CommentCount,
		StatusUpdateTime: r.StatusUpdateTime,
		UpvoteTime:       r.UpvoteTime,
	}
}
