package dto

import "time"

// ──────────────────────────────────────────
// Poll requests
// ──────────────────────────────────────────

type PollOptionInput struct {
	Text string `json:"text" validate:"required,min=1,max=100"`
}

type CreatePollRequest struct {
	TopicID          int               `json:"topic_id" validate:"required,min=1"`
	Title            string            `json:"title" validate:"required,min=1,max=100"`
	Description      string            `json:"description" validate:"max=500"`
	Type             string            `json:"type" validate:"required,oneof=single multiple"`
	MinChoice        int               `json:"min_choice" validate:"min=1"`
	MaxChoice        int               `json:"max_choice" validate:"min=1"`
	Deadline         *string           `json:"deadline"`
	ResultVisibility string            `json:"result_visibility" validate:"required,oneof=always after_vote after_deadline"`
	IsAnonymous      bool              `json:"is_anonymous"`
	CanChangeVote    bool              `json:"can_change_vote"`
	Options          []PollOptionInput `json:"options" validate:"required,min=2,max=20"`
}

type UpdatePollRequest struct {
	PollID           int               `json:"poll_id" validate:"required,min=1"`
	Title            string            `json:"title" validate:"required,min=1,max=100"`
	Description      string            `json:"description" validate:"max=500"`
	Type             string            `json:"type" validate:"required,oneof=single multiple"`
	MinChoice        int               `json:"min_choice" validate:"min=1"`
	MaxChoice        int               `json:"max_choice" validate:"min=1"`
	Deadline         *string           `json:"deadline"`
	ResultVisibility string            `json:"result_visibility" validate:"required,oneof=always after_vote after_deadline"`
	IsAnonymous      bool              `json:"is_anonymous"`
	CanChangeVote    bool              `json:"can_change_vote"`
	Options          PollOptionsUpdate `json:"options"`
}

type PollOptionsUpdate struct {
	Add    []PollOptionInput       `json:"add"`
	Update []PollOptionUpdateInput `json:"update"`
	Delete []int                   `json:"delete"`
}

type PollOptionUpdateInput struct {
	OptionID int    `json:"option_id" validate:"required,min=1"`
	Text     string `json:"text" validate:"required,min=1,max=100"`
}

type VoteRequest struct {
	PollID        int   `json:"poll_id" validate:"required,min=1"`
	OptionIDArray []int `json:"option_id_array" validate:"required,min=1"`
}

type GetPollByTopicRequest struct {
	TopicID int `query:"topic_id" validate:"required,min=1"`
}

type GetPollLogRequest struct {
	PollID int `query:"poll_id" validate:"required,min=1"`
	Page   int `query:"page" validate:"min=1"`
	Limit  int `query:"limit" validate:"min=1,max=50"`
}

// ──────────────────────────────────────────
// Poll responses
// ──────────────────────────────────────────

type PollOptionResponse struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	VoteCount *int   `json:"vote_count"`
	IsVoted   bool   `json:"is_voted"`
}

type TopicPollResponse struct {
	ID               int                  `json:"id"`
	Title            string               `json:"title"`
	Description      string               `json:"description"`
	MinChoice        int                  `json:"min_choice"`
	MaxChoice        int                  `json:"max_choice"`
	Deadline         *time.Time           `json:"deadline"`
	Type             string               `json:"type"`
	Status           string               `json:"status"`
	ResultVisibility string               `json:"result_visibility"`
	IsAnonymous      bool                 `json:"is_anonymous"`
	CanChangeVote    bool                 `json:"can_change_vote"`
	TopicID          int                  `json:"topic_id"`
	Created          time.Time            `json:"created"`
	Updated          time.Time            `json:"updated"`
	User             KunUser              `json:"user"`
	Options          []PollOptionResponse `json:"option"`
	HasVoted         bool                 `json:"has_voted"`
	Voters           []KunUser            `json:"voters"`
	VotersCount      int                  `json:"voters_count"`
	VoteCount        *int                 `json:"vote_count"`
}

type PollVoteLogEntry struct {
	ID      int       `json:"id"`
	Created time.Time `json:"created"`
	User    KunUser   `json:"user"`
	Option  string    `json:"option"`
}
