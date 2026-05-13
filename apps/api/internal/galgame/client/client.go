package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"kun-galgame-api/pkg/errors"
)

// GalgameClient calls the Galgame Wiki Service via HTTP.
//
// Holds two authentication contexts:
//   - per-request Bearer token (forwarded from the user's kungal session) —
//     for user-identity endpoints like submit / claim / patch-draft;
//   - a pre-built HTTP Basic header (OAuth client_id:secret, reused from
//     pkg/userclient credentials per decision 3 in 07-submission docs) —
//     for service-to-service endpoints like /galgame/messages/feed.
type GalgameClient struct {
	baseURL    string
	httpClient *http.Client
	basicAuth  string
}

// NewGalgameClient builds a client that can only do anonymous + Bearer calls.
// Suitable when service-to-service endpoints aren't needed.
func NewGalgameClient(baseURL string) *GalgameClient {
	return &GalgameClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewGalgameClientWithBasicAuth additionally enables service-to-service
// endpoints (currently: /galgame/messages/feed) by pre-computing the Basic
// auth header. Pass the same OAuth Client ID/secret used by pkg/userclient.
func NewGalgameClientWithBasicAuth(baseURL, clientID, clientSecret string) *GalgameClient {
	c := NewGalgameClient(baseURL)
	c.basicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))
	return c
}

// apiResponse is the standard {code, message, data} wrapper.
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Get performs a GET request to the wiki service.
func (c *GalgameClient) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, *errors.AppError) {
	return c.GetWithToken(ctx, path, "", query)
}

// GetWithToken is like Get but attaches a Bearer token. Used by endpoints
// whose response shape depends on the caller's identity:
//   - /galgame/batch with Bearer returns the caller's own pending drafts
//   - /galgame/search?include_pending=true returns the caller's pending hits
//   - /galgame/mine and /galgame/messages/mine are inherently user-scoped
//
// token "" reduces to an anonymous GET (same as Get).
func (c *GalgameClient) GetWithToken(ctx context.Context, path, token string, query url.Values) (json.RawMessage, *errors.AppError) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.doRequest(req)
}

// PostWithToken performs a POST with Bearer token.
//
// contentType controls how body is forwarded:
//   - "" (empty)        → defaults to "application/json"; struct/map bodies
//                         are JSON-marshaled
//   - "application/json" → same as empty
//   - any multipart/* / form-encoded / etc. → body MUST be passed as
//                                              []byte / json.RawMessage,
//                                              forwarded byte-for-byte
//                                              with the boundary preserved
func (c *GalgameClient) PostWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "POST", path, token, body, contentType)
}

// PutWithToken performs a PUT with Bearer token. See PostWithToken for
// contentType semantics.
func (c *GalgameClient) PutWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "PUT", path, token, body, contentType)
}

// DeleteWithToken performs a DELETE with Bearer token. See PostWithToken
// for contentType semantics.
func (c *GalgameClient) DeleteWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "DELETE", path, token, body, contentType)
}

// PatchWithToken performs a PATCH with Bearer token. Used by user draft
// edits (PATCH /galgame/:gid for status IN (3,4)). See PostWithToken for
// contentType semantics.
func (c *GalgameClient) PatchWithToken(ctx context.Context, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "PATCH", path, token, body, contentType)
}

func (c *GalgameClient) mutateWithToken(ctx context.Context, method, path, token string, body any, contentType string) (json.RawMessage, *errors.AppError) {
	if contentType == "" {
		contentType = "application/json"
	}

	var bodyReader io.Reader
	if body != nil {
		// Pass-through for already-encoded bodies (multipart, form-urlencoded,
		// etc.). Without this, json.Marshal would wrap raw bytes in quotes
		// and lose the multipart boundary.
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case json.RawMessage:
			bodyReader = bytes.NewReader([]byte(v))
		default:
			b, err := json.Marshal(body)
			if err != nil {
				return nil, errors.ErrInternal("序列化请求失败")
			}
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.doRequest(req)
}

// WikiUserStats is the user galgame stats from wiki service.
type WikiUserStats struct {
	GalgameCreated      int64 `json:"galgame_created"`
	GalgameCreatedToday int64 `json:"galgame_created_today"`
	GalgameContributed  int64 `json:"galgame_contributed"`
}

// GetUserStats fetches galgame-related stats for a user from wiki.
func (c *GalgameClient) GetUserStats(ctx context.Context, uid int) (*WikiUserStats, error) {
	path := fmt.Sprintf("/galgame/user/%d/stats", uid)
	data, appErr := c.Get(ctx, path, nil)
	if appErr != nil {
		return nil, appErr
	}

	var stats WikiUserStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// WikiAdminStats is the admin stats response from wiki service.
type WikiAdminStats struct {
	Totals map[string]int64   `json:"totals"`
	Daily  []map[string]any   `json:"daily"`
}

// GetAdminStats fetches wiki-side admin stats for the last N days.
func (c *GalgameClient) GetAdminStats(ctx context.Context, days int) (*WikiAdminStats, error) {
	query := url.Values{"days": {fmt.Sprintf("%d", days)}}
	data, appErr := c.Get(ctx, "/admin/stats", query)
	if appErr != nil {
		return nil, appErr
	}

	var stats WikiAdminStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// GalgameBrief is the lightweight metadata returned by /galgame/batch.
//
// Status is meaningful for Bearer-authenticated calls (which can see the
// caller's own status=3 pending / 4 declined drafts in addition to status=0).
// Anonymous calls always get status=0 entries — see 01-galgame.md.
//
// BannerImageHash is the image_service content hash. Frontend rendering
// prefers it (resolveBannerUrl(hash)) and falls back to the legacy Banner
// URL when the hash is empty.
type GalgameBrief struct {
	ID                 int    `json:"id"`
	VndbID             string `json:"vndb_id"`
	NameEnUs           string `json:"name_en_us"`
	NameJaJp           string `json:"name_ja_jp"`
	NameZhCn           string `json:"name_zh_cn"`
	NameZhTw           string `json:"name_zh_tw"`
	Banner             string `json:"banner"`
	BannerImageHash    string `json:"banner_image_hash"`
	Status             int    `json:"status"`
	ContentLimit       string `json:"content_limit"`
	UserID             int    `json:"user_id"`
	ResourceUpdateTime string `json:"resource_update_time"`
	OriginalLanguage   string `json:"original_language"`
	AgeLimit           string `json:"age_limit"`
}

// GetBatch fetches lightweight galgame info for multiple IDs anonymously
// (status=0 only). Returns a map[galgameID] -> GalgameBrief for easy lookup.
//
// For "show me my own pending drafts too" use GetBatchWithViewer.
func (c *GalgameClient) GetBatch(ctx context.Context, ids []int) (map[int]GalgameBrief, *errors.AppError) {
	return c.GetBatchWithViewer(ctx, ids, "")
}

// GetBatchWithViewer is the Bearer-aware batch fetch. With a non-empty token
// the wiki additionally returns any status=3/4 row whose user_id matches
// the JWT's uid claim — used by the "我的提交"/"发布向导" UX.
//
// token="" reduces to the anonymous form.
func (c *GalgameClient) GetBatchWithViewer(ctx context.Context, ids []int, token string) (map[int]GalgameBrief, *errors.AppError) {
	if len(ids) == 0 {
		return map[int]GalgameBrief{}, nil
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.Itoa(id)
	}
	query := url.Values{"ids": {joinStrings(idStrs, ",")}}

	data, appErr := c.GetWithToken(ctx, "/galgame/batch", token, query)
	if appErr != nil {
		return nil, appErr
	}

	var briefs []GalgameBrief
	if err := json.Unmarshal(data, &briefs); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 批量响应失败")
	}

	result := make(map[int]GalgameBrief, len(briefs))
	for _, b := range briefs {
		result[b.ID] = b
	}
	return result, nil
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

// ──────────────────────────────────────────
// Submission (user-identity, Bearer-forwarded)
// ──────────────────────────────────────────

// SubmitDraft posts a new pending submission (status=3). Returns the wiki
// response data raw so the handler can forward verbatim. See
// docs/galgame_wiki/07-submission.md §POST /galgame/submit.
func (c *GalgameClient) SubmitDraft(ctx context.Context, token string, body []byte, contentType string) (json.RawMessage, *errors.AppError) {
	return c.PostWithToken(ctx, "/galgame/submit", token, json.RawMessage(body), contentType)
}

// ClaimDraft flips a VNDB-source draft (status=2) to published (status=0)
// and assigns the caller as creator + contributor. Server enforces the
// status precondition.
func (c *GalgameClient) ClaimDraft(ctx context.Context, token string, gid int) (json.RawMessage, *errors.AppError) {
	path := "/galgame/" + strconv.Itoa(gid) + "/claim"
	return c.PostWithToken(ctx, path, token, json.RawMessage(`{}`), "application/json")
}

// PatchDraft updates the caller's own pending/declined draft (status IN 3,4).
// If the row was status=4, the wiki flips it back to status=3 (re-queues).
func (c *GalgameClient) PatchDraft(ctx context.Context, token string, gid int, body []byte, contentType string) (json.RawMessage, *errors.AppError) {
	path := "/galgame/" + strconv.Itoa(gid)
	return c.PatchWithToken(ctx, path, token, json.RawMessage(body), contentType)
}

// DeleteDraft hard-deletes the caller's own pending/declined draft. Wiki
// CASCADEs the associated wiki tables; kungal still needs to clean its
// local stub if interaction lazy-created one.
func (c *GalgameClient) DeleteDraft(ctx context.Context, token string, gid int) *errors.AppError {
	path := "/galgame/" + strconv.Itoa(gid)
	_, err := c.DeleteWithToken(ctx, path, token, nil, "")
	return err
}

// ──────────────────────────────────────────
// Message feed (service identity, Basic auth)
// ──────────────────────────────────────────

// WikiMessageGalgameBrief is the brief embed inside each WikiMessage.
// Null on hard-deleted galgames — consumers must null-check.
type WikiMessageGalgameBrief struct {
	ID     int `json:"id"`
	Status int `json:"status"`
}

// WikiMessage matches the per-message shape in /galgame/messages/feed.
// See docs/galgame_wiki/08-messages.md for the wire format.
type WikiMessage struct {
	ID           int64                    `json:"id"`
	Type         string                   `json:"type"`
	GalgameID    int                      `json:"galgame_id"`
	Galgame      *WikiMessageGalgameBrief `json:"galgame"`
	ActorUserID  int                      `json:"actor_user_id"`
	TargetUserID *int                     `json:"target_user_id"`
	Payload      json.RawMessage          `json:"payload"`
	CreatedAt    string                   `json:"created_at"`
}

// WikiMessageFeed is the envelope returned by /galgame/messages/feed.
type WikiMessageFeed struct {
	Items   []WikiMessage `json:"items"`
	HasMore bool          `json:"has_more"`
}

// MessagesFeed pulls a batch of admin-triggered events (approved /
// declined / banned / unbanned) using OAuth Client Basic Auth. Used by
// the wiki-message sync cron. Returns ErrInternal if the client wasn't
// constructed with NewGalgameClientWithBasicAuth.
func (c *GalgameClient) MessagesFeed(ctx context.Context, sinceID int64, limit int) (*WikiMessageFeed, *errors.AppError) {
	if c.basicAuth == "" {
		return nil, errors.ErrInternal("wiki client 未配置 Basic Auth 凭证")
	}
	if limit <= 0 {
		limit = 1000
	}

	reqURL := c.baseURL + "/galgame/messages/feed?since_id=" +
		strconv.FormatInt(sinceID, 10) + "&limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Authorization", c.basicAuth)

	data, appErr := c.doRequest(req)
	if appErr != nil {
		return nil, appErr
	}
	var feed WikiMessageFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, errors.ErrInternal("解析 wiki 消息 feed 失败")
	}
	return &feed, nil
}

func (c *GalgameClient) doRequest(req *http.Request) (json.RawMessage, *errors.AppError) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.ErrInternal(fmt.Sprintf("Wiki 服务请求失败: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.ErrInternal("读取 Wiki 响应失败")
	}

	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	if result.Code != 0 {
		// Transparently forward wiki service error code + message
		return nil, errors.New(result.Code, result.Message, resp.StatusCode)
	}

	return result.Data, nil
}
