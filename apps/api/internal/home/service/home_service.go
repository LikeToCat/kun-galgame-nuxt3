package service

import (
	"context"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/home/dto"
	"kun-galgame-api/internal/home/repository"
)

const (
	homeGalgameLimit = 12
	homeTopicLimit   = 10
)

type HomeService struct {
	repo   *repository.HomeRepository
	wikiGC *galgameClient.GalgameClient
}

func NewHomeService(
	repo *repository.HomeRepository,
	gc *galgameClient.GalgameClient,
) *HomeService {
	return &HomeService{repo: repo, wikiGC: gc}
}

// GetHome returns homepage data: galgames + topics.
func (s *HomeService) GetHome(ctx context.Context, isSFW bool) (*dto.HomeResponse, error) {
	galgames, err := s.getHomeGalgames(ctx, isSFW)
	if err != nil {
		return nil, err
	}
	topics, err := s.getHomeTopics(isSFW)
	if err != nil {
		return nil, err
	}
	return &dto.HomeResponse{Galgames: galgames, Topics: topics}, nil
}

func (s *HomeService) getHomeGalgames(ctx context.Context, isSFW bool) ([]dto.HomeGalgame, error) {
	// Step 1: Get galgame IDs sorted by local interaction data
	localRows, err := s.repo.FindRecentGalgames(homeGalgameLimit)
	if err != nil {
		return nil, err
	}
	if len(localRows) == 0 {
		return []dto.HomeGalgame{}, nil
	}

	// Step 2: Batch fetch metadata from wiki
	galgameIDs := make([]int, len(localRows))
	for i, r := range localRows {
		galgameIDs[i] = r.ID
	}
	briefMap, appErr := s.wikiGC.GetBatch(ctx, galgameIDs)
	if appErr != nil {
		return nil, appErr
	}

	// Step 3: Batch fetch users
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	users := s.repo.FindUsersByIDs(userIDs)
	userMap := make(map[int]repository.UserInfoRow, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Step 4: Batch fetch platforms/languages from local galgame_resource
	resources := s.repo.FindResourcePlatformLanguage(galgameIDs)
	platformMap := map[int]map[string]bool{}
	languageMap := map[int]map[string]bool{}
	for _, r := range resources {
		if platformMap[r.GalgameID] == nil {
			platformMap[r.GalgameID] = map[string]bool{}
		}
		if languageMap[r.GalgameID] == nil {
			languageMap[r.GalgameID] = map[string]bool{}
		}
		platformMap[r.GalgameID][r.Platform] = true
		languageMap[r.GalgameID][r.Language] = true
	}

	// Step 5: Assemble results in original order
	result := make([]dto.HomeGalgame, 0, len(localRows))
	for _, lr := range localRows {
		b, ok := briefMap[lr.ID]
		if !ok {
			continue // wiki doesn't have this galgame
		}
		if isSFW && b.ContentLimit != "sfw" {
			continue
		}
		u := userMap[b.UserID]
		result = append(result, dto.HomeGalgame{
			ID: lr.ID,
			Name: dto.LocaleName{
				EnUS: b.NameEnUs, JaJP: b.NameJaJp,
				ZhCN: b.NameZhCn, ZhTW: b.NameZhTw,
			},
			Banner:             b.Banner,
			User:               dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ContentLimit:       b.ContentLimit,
			View:               lr.View,
			LikeCount:          lr.LikeCount,
			ResourceUpdateTime: b.ResourceUpdateTime,
			Platform:           mapKeys(platformMap[lr.ID]),
			Language:           mapKeys(languageMap[lr.ID]),
		})
	}

	return result, nil
}

func (s *HomeService) getHomeTopics(isSFW bool) ([]dto.HomeTopic, error) {
	rows, err := s.repo.FindHomeTopics(homeTopicLimit, isSFW)
	if err != nil {
		return nil, err
	}

	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	sections := s.repo.FindTopicSections(topicIDs)
	sectionMap := map[int][]string{}
	for _, sct := range sections {
		sectionMap[sct.TopicID] = append(sectionMap[sct.TopicID], sct.SectionName)
	}

	tags := s.repo.FindTopicTags(topicIDs)
	tagMap := map[int][]string{}
	for _, t := range tags {
		tagMap[t.TopicID] = append(tagMap[t.TopicID], t.TagName)
	}

	result := make([]dto.HomeTopic, len(rows))
	for i, r := range rows {
		topicTags := tagMap[r.ID]
		if topicTags == nil {
			topicTags = []string{}
		}
		topicSections := sectionMap[r.ID]
		if topicSections == nil {
			topicSections = []string{}
		}

		result[i] = dto.HomeTopic{
			ID:               r.ID,
			Title:            r.Title,
			View:             r.View,
			LikeCount:        r.LikeCount,
			ReplyCount:       r.ReplyCount,
			CommentCount:     r.CommentCount,
			HasBestAnswer:    r.BestAnswerID != nil,
			IsPollTopic:      false,
			IsNSFWTopic:      r.IsNSFW,
			Section:          topicSections,
			Tag:              topicTags,
			User:             dto.UserBrief{ID: r.UserID, Name: r.UserName, Avatar: r.UserAvatar},
			Status:           r.Status,
			UpvoteTime:       r.UpvoteTime,
			StatusUpdateTime: r.StatusUpdateTime,
		}
	}

	return result, nil
}

func mapKeys(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
