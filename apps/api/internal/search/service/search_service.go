package service

import (
	"strings"

	"kun-galgame-api/internal/search/dto"
	"kun-galgame-api/internal/search/repository"
	"kun-galgame-api/pkg/errors"
)

type SearchService struct {
	repo *repository.SearchRepository
}

func NewSearchService(repo *repository.SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

// tokenize splits a keyword string into trimmed non-empty tokens.
// Returns an error if the result is empty.
func tokenize(raw string) ([]string, *errors.AppError) {
	keywords := strings.Fields(strings.TrimSpace(raw))
	if len(keywords) == 0 {
		return nil, errors.ErrBadRequest("搜索关键词不能为空")
	}
	return keywords, nil
}

// SearchTopics returns topic search results.
func (s *SearchService) SearchTopics(raw string, page, limit int) (*dto.PaginatedResult[dto.TopicItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchTopics(keywords, page, limit)

	items := make([]dto.TopicItem, len(rows))
	for i, r := range rows {
		items[i] = dto.TopicItem{
			ID: r.ID, Title: r.Title, View: r.View, Status: r.Status,
			LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			CommentCount: r.CommentCount, StatusUpdateTime: r.StatusUpdateTime,
			User: dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
		}
	}
	return &dto.PaginatedResult[dto.TopicItem]{Items: items, Total: total}, nil
}

// SearchUsers returns user search results.
func (s *SearchService) SearchUsers(raw string, page, limit int) (*dto.PaginatedResult[dto.UserItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchUsers(keywords, page, limit)

	items := make([]dto.UserItem, len(rows))
	for i, r := range rows {
		items[i] = dto.UserItem{
			ID: r.ID, Name: r.Name, Avatar: r.Avatar, Bio: r.Bio,
			Moemoepoint: r.Moemoepoint, Created: r.Created,
		}
	}
	return &dto.PaginatedResult[dto.UserItem]{Items: items, Total: total}, nil
}

// SearchReplies returns reply search results.
func (s *SearchService) SearchReplies(raw string, page, limit int) (*dto.PaginatedResult[dto.ReplyItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchReplies(keywords, page, limit)

	items := make([]dto.ReplyItem, len(rows))
	for i, r := range rows {
		items[i] = dto.ReplyItem{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content, Floor: r.Floor,
			User:    dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Created: r.Created,
		}
	}
	return &dto.PaginatedResult[dto.ReplyItem]{Items: items, Total: total}, nil
}

// SearchComments returns comment search results.
func (s *SearchService) SearchComments(raw string, page, limit int) (*dto.PaginatedResult[dto.CommentItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchComments(keywords, page, limit)

	items := make([]dto.CommentItem, len(rows))
	for i, r := range rows {
		items[i] = dto.CommentItem{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content,
			User:    dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Created: r.Created,
		}
	}
	return &dto.PaginatedResult[dto.CommentItem]{Items: items, Total: total}, nil
}
