package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"kun-galgame-api/pkg/errors"
)

// GalgameClient calls the Galgame Wiki Service via HTTP.
type GalgameClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewGalgameClient(baseURL string) *GalgameClient {
	return &GalgameClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// apiResponse is the standard {code, message, data} wrapper.
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Get performs a GET request to the wiki service.
func (c *GalgameClient) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, *errors.AppError) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
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
type GalgameBrief struct {
	ID                 int    `json:"id"`
	VndbID             string `json:"vndb_id"`
	NameEnUs           string `json:"name_en_us"`
	NameJaJp           string `json:"name_ja_jp"`
	NameZhCn           string `json:"name_zh_cn"`
	NameZhTw           string `json:"name_zh_tw"`
	Banner             string `json:"banner"`
	ContentLimit       string `json:"content_limit"`
	UserID             int    `json:"user_id"`
	ResourceUpdateTime string `json:"resource_update_time"`
	OriginalLanguage   string `json:"original_language"`
	AgeLimit           string `json:"age_limit"`
}

// GetBatch fetches lightweight galgame info for multiple IDs.
// Returns a map[galgameID] -> GalgameBrief for easy lookup.
func (c *GalgameClient) GetBatch(ctx context.Context, ids []int) (map[int]GalgameBrief, *errors.AppError) {
	if len(ids) == 0 {
		return map[int]GalgameBrief{}, nil
	}

	// Build comma-separated IDs
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	query := url.Values{"ids": {joinStrings(idStrs, ",")}}

	data, appErr := c.Get(ctx, "/galgame/batch", query)
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
