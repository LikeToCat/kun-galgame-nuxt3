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
func (c *GalgameClient) PostWithToken(ctx context.Context, path, token string, body any) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "POST", path, token, body)
}

// PutWithToken performs a PUT with Bearer token.
func (c *GalgameClient) PutWithToken(ctx context.Context, path, token string, body any) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "PUT", path, token, body)
}

// DeleteWithToken performs a DELETE with Bearer token.
func (c *GalgameClient) DeleteWithToken(ctx context.Context, path, token string, body any) (json.RawMessage, *errors.AppError) {
	return c.mutateWithToken(ctx, "DELETE", path, token, body)
}

func (c *GalgameClient) mutateWithToken(ctx context.Context, method, path, token string, body any) (json.RawMessage, *errors.AppError) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.ErrInternal("序列化请求失败")
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.doRequest(req)
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
