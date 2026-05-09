package service

import (
	"context"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/dto"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"
)

// ──────────────────────────────────────────
// Poll mappers
// ──────────────────────────────────────────

// buildPollResponse assembles a TopicPollResponse from a poll model and the
// associated option/voter data loaded via the repository. It does not perform
// any DB writes; callers pass in the logged-in user context via uid/role.
// Identity for voters/creator is hydrated from OAuth via userclient.
func (s *PollService) buildPollResponse(ctx context.Context, poll *topicModel.TopicPoll, uid, role int) dto.TopicPollResponse {
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
			voterIDs, _ := s.pollRepo.FindDistinctVoterIDs(poll.ID, 5)
			vmap := s.userClient.Hydrate(ctx, voterIDs)
			voters = make([]dto.KunUser, 0, len(voterIDs))
			for _, id := range voterIDs {
				u := vmap[id]
				if !userclient.IsRenderable(u) {
					continue
				}
				voters = append(voters, dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
			}
		}
		vc, _ := s.pollRepo.CountDistinctVoters(poll.ID)
		votersCount = vc
		tc, _ := s.pollRepo.CountTotalVotes(poll.ID)
		totalVoteCount = &tc
	}
	if voters == nil {
		voters = []dto.KunUser{}
	}

	creatorU, _, _ := s.userClient.User(ctx, poll.UserID)
	if creatorU.ID == 0 {
		creatorU = userclient.Placeholder(poll.UserID)
	}

	return dto.TopicPollResponse{
		ID: poll.ID, Title: poll.Title, Description: poll.Description,
		MinChoice: poll.MinChoice, MaxChoice: poll.MaxChoice,
		Deadline: poll.Deadline, Type: poll.Type, Status: poll.Status,
		ResultVisibility: poll.ResultVisibility,
		IsAnonymous:      poll.IsAnonymous, CanChangeVote: poll.CanChangeVote,
		TopicID: poll.TopicID, Created: poll.CreatedAt, Updated: poll.UpdatedAt,
		User:    dto.KunUser{ID: creatorU.ID, Name: creatorU.Name, Avatar: creatorU.Avatar},
		Options: optionResponses,
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
// Fetches targets/comments/like-status via the repository in bulk. Identity
// (name/avatar/moemoepoint) is hydrated from OAuth+kungal_user_state since
// the repo no longer joins on the user table.
func (s *ReplyService) buildReplyResponses(
	ctx context.Context,
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

	// Collect every uid we'll render: reply authors + target authors + comment
	// authors + comment target authors. Hydrate in one batch.
	uidSet := make(map[int]struct{})
	for _, r := range rows {
		uidSet[r.UserID] = struct{}{}
	}
	for _, ts := range targetMap {
		for _, t := range ts {
			uidSet[t.TargetUserID] = struct{}{}
		}
	}
	for _, cs := range commentMap {
		for _, c := range cs {
			uidSet[c.UserID] = struct{}{}
			if c.TargetUserID > 0 {
				uidSet[c.TargetUserID] = struct{}{}
			}
		}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		if id > 0 {
			uids = append(uids, id)
		}
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	// moemoepoint comes from kungal_user_state, not OAuth.
	moeMap := make(map[int]int, len(rows))
	for _, r := range rows {
		if _, seen := moeMap[r.UserID]; seen {
			continue
		}
		if state, _ := s.stateRepo.FindByID(r.UserID); state != nil {
			moeMap[r.UserID] = state.Moemoepoint
		}
	}

	kunUser := func(id int) dto.KunUser {
		u := userMap[id]
		return dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}

	responses := make([]dto.TopicReplyResponse, 0, len(rows))
	for _, r := range rows {
		// Drop banned authors entirely.
		if u, ok := userMap[r.UserID]; ok && !userclient.IsRenderable(u) {
			continue
		}
		var targets []dto.ReplyTargetResponse
		if ts, ok := targetMap[r.ID]; ok {
			for _, t := range ts {
				preview := truncate(t.TargetContent, 150)
				targets = append(targets, dto.ReplyTargetResponse{
					ID:                   t.TargetReplyID,
					Floor:                t.TargetFloor,
					User:                 kunUser(t.TargetUserID),
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
					User:       kunUser(c.UserID),
					TargetUser: kunUser(c.TargetUserID),
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

		author := userMap[r.UserID]
		responses = append(responses, dto.TopicReplyResponse{
			ID:      r.ID,
			TopicID: r.TopicID,
			Floor:   r.Floor,
			User: dto.KunUserWithMoemoepoint{
				ID:          author.ID,
				Name:        author.Name,
				Avatar:      author.Avatar,
				Moemoepoint: moeMap[r.UserID],
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
		})
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
