package service

import (
	"context"
	"log/slog"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/home/dto"
	"kun-galgame-api/internal/home/repository"
	"kun-galgame-api/pkg/userclient"
)

const (
	// Public count — the number of galgame cards the homepage actually renders.
	homeGalgameLimit = 12
	// Over-fetch factor: wiki may be missing a subset of local IDs (migration
	// lag) and the SFW filter drops some too, so we need slack above
	// homeGalgameLimit to reliably return `homeGalgameLimit` usable cards.
	homeGalgameFetchLimit = 24
	homeTopicLimit        = 10
)

type HomeService struct {
	repo       *repository.HomeRepository
	wikiGC     *galgameClient.GalgameClient
	userClient *userclient.Client
}

func NewHomeService(
	repo *repository.HomeRepository,
	gc *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *HomeService {
	return &HomeService{repo: repo, wikiGC: gc, userClient: userClient}
}

// GetHome returns homepage data: galgames + topics.
//
// Galgame data is best-effort: if the wiki service is unreachable we log and
// return an empty galgame list rather than failing the whole endpoint, so the
// topic feed still renders. Topic lookup failures are still propagated — they
// come from the local DB and indicate something more serious.
func (s *HomeService) GetHome(ctx context.Context, isSFW bool) (*dto.HomeResponse, error) {
	galgames, err := s.getHomeGalgames(ctx, isSFW)
	if err != nil {
		slog.Warn("首页 galgame 获取失败, 降级为空列表", "error", err)
		galgames = []dto.HomeGalgame{}
	}
	topics, err := s.getHomeTopics(ctx, isSFW)
	if err != nil {
		return nil, err
	}
	return &dto.HomeResponse{Galgames: galgames, Topics: topics}, nil
}

func (s *HomeService) getHomeGalgames(ctx context.Context, isSFW bool) ([]dto.HomeGalgame, error) {
	// Step 1: Over-fetch local rows. Wiki may not know some IDs (migration
	// lag) and the SFW filter below also drops rows, so we pull 2x the
	// displayed count to reliably land at `homeGalgameLimit` after filtering.
	localRows, err := s.repo.FindRecentGalgames(homeGalgameFetchLimit)
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

	// Step 3: Batch fetch users from OAuth
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

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

	// Step 5: Assemble results in original order, stopping at homeGalgameLimit.
	result := make([]dto.HomeGalgame, 0, homeGalgameLimit)
	for _, lr := range localRows {
		if len(result) >= homeGalgameLimit {
			break
		}
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

func (s *HomeService) getHomeTopics(ctx context.Context, isSFW bool) ([]dto.HomeTopic, error) {
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

	uids := userclient.CollectIDs(rows, func(r repository.TopicRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	result := make([]dto.HomeTopic, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		topicTags := tagMap[r.ID]
		if topicTags == nil {
			topicTags = []string{}
		}
		topicSections := sectionMap[r.ID]
		if topicSections == nil {
			topicSections = []string{}
		}

		result = append(result, dto.HomeTopic{
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
			User:             dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Status:           r.Status,
			UpvoteTime:       r.UpvoteTime,
			StatusUpdateTime: r.StatusUpdateTime,
		})
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
