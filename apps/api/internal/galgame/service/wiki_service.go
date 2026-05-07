package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
)

// WikiService handles pass-through proxying to the wiki service and the
// common "wiki + local user resolution" pattern used by galgame sub-routes.
type WikiService struct {
	wikiClient  *client.GalgameClient
	galgameRepo *repository.GalgameRepository
}

func NewWikiService(
	wikiClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
) *WikiService {
	return &WikiService{wikiClient: wikiClient, galgameRepo: galgameRepo}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// ProxyGet forwards a GET request to the wiki service and returns the raw body.
// The gateway path is translated to the wiki path via client.ToWikiPath.
func (s *WikiService) ProxyGet(
	ctx context.Context,
	gatewayPath string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.Get(ctx, client.ToWikiPath(gatewayPath), query)
}

// ProxyWrite forwards a write request (POST/PUT/DELETE) with OAuth token.
//
// contentType is the original Content-Type from the gateway request — kept
// verbatim so multipart boundaries / form-encoded payloads survive the hop.
// Empty defaults to application/json (what the wiki expects on JSON bodies).
func (s *WikiService) ProxyWrite(
	ctx context.Context,
	method, gatewayPath, token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	wikiPath := client.ToWikiPath(gatewayPath)
	payload := json.RawMessage(body)

	switch method {
	case "POST":
		return s.wikiClient.PostWithToken(ctx, wikiPath, token, payload, contentType)
	case "PUT":
		return s.wikiClient.PutWithToken(ctx, wikiPath, token, payload, contentType)
	case "DELETE":
		return s.wikiClient.DeleteWithToken(ctx, wikiPath, token, payload, contentType)
	default:
		return nil, errors.ErrBadRequest("不支持的方法")
	}
}

// ──────────────────────────────────────────
// Galgame Links
// ──────────────────────────────────────────

type wikiLinkRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Link      string `json:"link"`
	GalgameID int    `json:"galgame_id"`
	UserID    int    `json:"user_id"`
}

// GetGalgameLinks fetches the links for a galgame from wiki and resolves the
// author user in the local DB.
func (s *WikiService) GetGalgameLinks(
	ctx context.Context,
	gid string,
) ([]dto.GalgameLink, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/links", nil)
	if appErr != nil {
		return nil, appErr
	}

	var rows []wikiLinkRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return []dto.GalgameLink{}, nil
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.UserID
	}
	userMap := s.galgameRepo.FindUsersByIDs(ids)

	out := make([]dto.GalgameLink, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameLink{
			ID:        r.ID,
			User:      userBriefToDTO(userMap[r.UserID]),
			GalgameID: r.GalgameID,
			Name:      r.Name,
			Link:      r.Link,
		}
	}
	return out, nil
}

// ──────────────────────────────────────────
// Galgame History (revisions)
// ──────────────────────────────────────────

type wikiRevisionRow struct {
	ID        int    `json:"id"`
	GalgameID int    `json:"galgame_id"`
	Revision  int    `json:"revision"`
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Note      string `json:"note"`
	IsMinor   bool   `json:"is_minor"`
	Created   string `json:"created"`
}

type wikiRevisionListResp struct {
	Items []wikiRevisionRow `json:"items"`
	Total int64             `json:"total"`
}

// GetGalgameHistory fetches revision history for a galgame and resolves users.
func (s *WikiService) GetGalgameHistory(
	ctx context.Context,
	gid string,
	query url.Values,
) (*dto.GalgameRevisionListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/revisions", query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiRevisionListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return &dto.GalgameRevisionListPage{Items: []dto.GalgameRevision{}, Total: 0}, nil
	}

	ids := make([]int, len(parsed.Items))
	for i, r := range parsed.Items {
		ids[i] = r.UserID
	}
	userMap := s.galgameRepo.FindUsersByIDs(ids)

	items := make([]dto.GalgameRevision, len(parsed.Items))
	for i, r := range parsed.Items {
		items[i] = dto.GalgameRevision{
			ID:       r.ID,
			Revision: r.Revision,
			Action:   r.Action,
			Note:     r.Note,
			User:     userBriefToDTO(userMap[r.UserID]),
			IsMinor:  r.IsMinor,
			Created:  r.Created,
		}
	}
	return &dto.GalgameRevisionListPage{Items: items, Total: parsed.Total}, nil
}

// ──────────────────────────────────────────
// Galgame PRs
// ──────────────────────────────────────────

type wikiPRRow struct {
	ID            int     `json:"id"`
	GalgameID     int     `json:"galgame_id"`
	Status        int     `json:"status"`
	Note          string  `json:"note"`
	BaseRevision  int     `json:"base_revision"`
	UserID        int     `json:"user_id"`
	CompletedTime *string `json:"completed_time"`
	Created       string  `json:"created"`
}

type wikiPRListResp struct {
	Items []wikiPRRow `json:"items"`
	Total int64       `json:"total"`
}

// GetGalgamePRs fetches the PR list for a galgame and resolves submitter users.
func (s *WikiService) GetGalgamePRs(
	ctx context.Context,
	gid string,
	query url.Values,
) (*dto.GalgamePRListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/prs", query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiPRListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return &dto.GalgamePRListPage{Items: []dto.GalgamePR{}, Total: 0}, nil
	}

	ids := make([]int, len(parsed.Items))
	for i, r := range parsed.Items {
		ids[i] = r.UserID
	}
	userMap := s.galgameRepo.FindUsersByIDs(ids)

	items := make([]dto.GalgamePR, len(parsed.Items))
	for i, r := range parsed.Items {
		items[i] = dto.GalgamePR{
			ID:            r.ID,
			GalgameID:     r.GalgameID,
			Status:        r.Status,
			Note:          r.Note,
			BaseRevision:  r.BaseRevision,
			User:          userBriefToDTO(userMap[r.UserID]),
			CompletedTime: r.CompletedTime,
			Created:       r.Created,
		}
	}
	return &dto.GalgamePRListPage{Items: items, Total: parsed.Total}, nil
}

