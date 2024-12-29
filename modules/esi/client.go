package esi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync/atomic"
	"time"

	"golang.org/x/oauth2"

	"github.com/guarzo/eveapi/common"
)

// EsiClient defines lower-level HTTP operations for ESI:
// handling Get/POST/DELETE, token refresh checks, caching, etc.
type EsiClient interface {
	GetJSON(ctx context.Context, endpoint string, entity interface{}, token *oauth2.Token, params map[string]string) error
	GetBytes(ctx context.Context, endpoint string, token *oauth2.Token, params map[string]string) ([]byte, error)
	PostJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error)
	DeleteJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error)
	DoRequest(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader, expectedStatus ...int) ([]byte, error)
}

// AuthClient is optional. If you want to do token refresh externally, define it here.
type AuthClient interface {
	RefreshToken(refreshToken string) (*oauth2.Token, error)
}

type esiClient struct {
	baseURL    string
	httpClient common.HttpClient
	cache      common.CacheRepository
	authClient AuthClient
}

// Some metrics counters (optional)
var (
	totalCalls    int64
	notFoundCount int64
	successCount  int64
	failCount     int64
)

// Default for how long to cache data. Adjust as needed.
const defaultCacheExpiration = 770 * time.Hour

// NewEsiClient creates a new EsiClient that will communicate with EVE ESI.
func NewEsiClient(baseURL string, httpClient common.HttpClient, cache common.CacheRepository, authClient AuthClient) EsiClient {
	return &esiClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		cache:      cache,
		authClient: authClient,
	}
}

// ---------------------------------------------------
// Implementation of EsiClient interface
// ---------------------------------------------------

// GetJSON retrieves JSON from an ESI endpoint and unmarshals into entity.
func (c *esiClient) GetJSON(ctx context.Context, endpoint string, entity interface{}, token *oauth2.Token, params map[string]string) error {
	data, err := c.GetBytes(ctx, endpoint, token, params)
	if err != nil {
		return err
	}
	return unmarshalJSON(data, entity)
}

// GetBytes retrieves raw bytes from an ESI endpoint, with caching if desired.
func (c *esiClient) GetBytes(ctx context.Context, endpoint string, token *oauth2.Token, params map[string]string) ([]byte, error) {
	if params == nil {
		params = map[string]string{}
	}
	// Example: set default datasource if not present
	if _, found := params["datasource"]; !found {
		params["datasource"] = "tranquility"
	}

	// build a cache key if you want to store the response
	cacheKey := c.buildCacheKey(endpoint, params)
	if cached, found := c.cache.Get(cacheKey); found {
		return cached, nil
	}

	urlStr, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, err
	}

	operation := func() (interface{}, error) {
		data, err := c.DoRequest(ctx, http.MethodGet, urlStr, token, nil)
		if err != nil {
			return nil, err
		}
		// store in cache
		c.cache.Set(cacheKey, data, defaultCacheExpiration)
		return data, nil
	}

	result, err := c.httpClient.RetryWithExponentialBackoff(operation)
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}

// PostJSON sends a POST with optional expected status codes.
func (c *esiClient) PostJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error) {
	urlStr, err := c.buildURL(endpoint, nil)
	if err != nil {
		return nil, err
	}
	return c.DoRequest(ctx, http.MethodPost, urlStr, token, body, expectedStatusCodes...)
}

// DeleteJSON sends a DELETE with optional expected status codes.
func (c *esiClient) DeleteJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error) {
	urlStr, err := c.buildURL(endpoint, nil)
	if err != nil {
		return nil, err
	}
	return c.DoRequest(ctx, http.MethodDelete, urlStr, token, body, expectedStatusCodes...)
}

// DoRequest is the core method that actually performs the HTTP request.
func (c *esiClient) DoRequest(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader, expectedStatus ...int) ([]byte, error) {
	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK}
	}

	// read the entire body so we can retry
	var bodyBytes []byte
	if body != nil {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		bodyBytes = b
	}

	// Execute request
	data, status, err := c.executeRequest(ctx, method, urlStr, token, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// if unauthorized/forbidden and we have refresh capability, try refresh
	if (status == http.StatusUnauthorized || status == http.StatusForbidden) && canRefresh(token, c.authClient) {
		newToken, refreshErr := c.authClient.RefreshToken(token.RefreshToken)
		if refreshErr == nil && newToken != nil {
			// retry with new token
			token = newToken
			data, status, err = c.executeRequest(ctx, method, urlStr, token, bytes.NewReader(bodyBytes))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("token refresh failed: %w", refreshErr)
		}
	}

	// metrics
	atomic.AddInt64(&totalCalls, 1)
	switch {
	case status == http.StatusNotFound:
		atomic.AddInt64(&notFoundCount, 1)
	case status >= 200 && status < 300:
		atomic.AddInt64(&successCount, 1)
	default:
		atomic.AddInt64(&failCount, 1)
	}

	if !statusMatches(status, expectedStatus) {
		return nil, &common.HTTPError{
			StatusCode: status,
			Body:       data,
		}
	}
	return data, nil
}

// executeRequest actually does the low-level HTTP
func (c *esiClient) executeRequest(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if token != nil && token.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %v", readErr)
	}
	return data, resp.StatusCode, nil
}

// buildURL merges baseURL + endpoint + params
func (c *esiClient) buildURL(endpoint string, params map[string]string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	path, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	fullURL := base.ResolveReference(path)
	q := fullURL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	fullURL.RawQuery = q.Encode()
	return fullURL.String(), nil
}

// build a cache key (optional usage)
func (c *esiClient) buildCacheKey(endpoint string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	queryParams := ""
	for _, k := range keys {
		queryParams += fmt.Sprintf("&%s=%s", k, params[k])
	}
	return fmt.Sprintf("esi:%s:%s", endpoint, queryParams)
}

func statusMatches(statusCode int, expected []int) bool {
	for _, s := range expected {
		if statusCode == s {
			return true
		}
	}
	return false
}

func canRefresh(token *oauth2.Token, auth AuthClient) bool {
	return token != nil && token.RefreshToken != "" && auth != nil
}

// unmarshalJSON helper
func unmarshalJSON(data []byte, out interface{}) error {
	return common.JSONUnmarshal(data, out)
}
