package zkill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
)

// ZKillClient is a lower-level interface for fetching from zKillboard’s API.
type ZKillClient interface {
	GetKillsPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
	GetLossPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
	RemoveCacheEntry(cacheKey string)
	GetSingleKillmail(ctx context.Context, killID int) (model.ZkillMailFeedResponse, error)
	BuildCacheKey(apiType, entityType string, entityID, year, month, page int) string
}

// zKillClient implements ZKillClient.
type zKillClient struct {
	BaseURL string
	Client  common.HttpClient
	Cache   common.CacheRepository
}

// NewZkillClient constructs a zKillClient. The baseURL is typically "https://zkillboard.com".
func NewZkillClient(baseURL string, client common.HttpClient, cache common.CacheRepository) ZKillClient {
	return &zKillClient{
		BaseURL: baseURL,
		Client:  client,
		Cache:   cache,
	}
}

const zkillCacheExpiration = 770 * time.Hour // Example expiration (~1 month)

// RemoveCacheEntry forcibly removes a specific cached entry.
func (zk *zKillClient) RemoveCacheEntry(cacheKey string) {
	zk.Cache.Delete(cacheKey)
}

// BuildCacheKey composes a string to store/fetch data in the CacheRepository.
func (zk *zKillClient) BuildCacheKey(apiType, entityType string, entityID, year, month, page int) string {
	// E.g. "zkill:kills:corporationID:9000000:2023:10:1"
	return fmt.Sprintf("zkill:%s:%sID:%d:%d:%02d:%d", apiType, entityType, entityID, year, month, page)
}

// GetKillsPageData fetches killmails (where entity is an attacker).
func (zk *zKillClient) GetKillsPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error) {
	return zk.fetchPageData(ctx, "kills", entityType, entityID, page, year, month)
}

// GetLossPageData fetches killmails (where entity is a victim).
func (zk *zKillClient) GetLossPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error) {
	return zk.fetchPageData(ctx, "losses", entityType, entityID, page, year, month)
}

// Private method that constructs the request URL and fetches data from zKillboard.
func (zk *zKillClient) fetchPageData(ctx context.Context, apiType, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error) {
	requestURL := fmt.Sprintf("%s/api/%s/%sID/%d/year/%d/month/%d/page/%d/",
		zk.BaseURL, apiType, entityType, entityID, year, month, page)
	cacheKey := zk.BuildCacheKey(apiType, entityType, entityID, year, month, page)

	// Decide if we should re-fetch if it’s the current month
	currentYear, currentMonth, _ := time.Now().Date()
	isCurrentMonth := (year == currentYear && month == int(currentMonth))

	// Try cache first
	if cachedData, found := zk.Cache.Get(cacheKey); found {
		var kills []model.ZkillMail
		if err := json.Unmarshal(cachedData, &kills); err == nil {
			return kills, nil
		}
	}

	// We either had no cache or invalid data. Make an HTTP GET request.
	kills, err := zk.doGetKillMails(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	// Maybe set a different expiration if it’s the current month. Adjust as you like.
	exp := zkillCacheExpiration
	if isCurrentMonth {
		exp = 24 * time.Hour // e.g. re-fetch more often
	}

	// Save result to cache
	bytes, err := json.Marshal(kills)
	if err == nil {
		zk.Cache.Set(cacheKey, bytes, exp)
	}

	return kills, nil
}

// doGetKillMails executes the actual HTTP request and decodes the JSON response.
func (zk *zKillClient) doGetKillMails(ctx context.Context, url string) ([]model.ZkillMail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := zk.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response from zKill: %d", resp.StatusCode)
	}

	var kills []model.ZkillMail
	if err = json.NewDecoder(resp.Body).Decode(&kills); err != nil {
		return nil, fmt.Errorf("failed to decode zkill JSON: %w", err)
	}
	return kills, nil
}

// -------------------------------------------------------------------------------------------
// NEW METHOD: GetSingleKillmail - fetch from /api/killID/<killID>/
// -------------------------------------------------------------------------------------------

// GetSingleKillmail fetches the single kill’s details from zKill at /api/killID/<killID>/.
// zKill normally returns an array of length 1 with the kill’s victim/attackers data.
func (zk *zKillClient) GetSingleKillmail(ctx context.Context, killID int) (model.ZkillMailFeedResponse, error) {
	// We'll define a specialized endpoint: /api/killID/<killID>/
	requestURL := fmt.Sprintf("%s/api/killID/%d/", zk.BaseURL, killID)

	// Construct a dedicated cache key for single kills
	cacheKey := fmt.Sprintf("zkill:single:killID:%d", killID)

	// Attempt to fetch from cache
	if cachedData, found := zk.Cache.Get(cacheKey); found {
		var kills []model.ZkillMailFeedResponse
		if err := json.Unmarshal(cachedData, &kills); err == nil && len(kills) > 0 {
			return kills[0], nil
		}
	}

	// If not in cache, fetch from zKill
	kills, err := zk.doGetSingleKillMails(ctx, requestURL)
	if err != nil {
		return model.ZkillMailFeedResponse{}, err
	}
	if len(kills) == 0 {
		return model.ZkillMailFeedResponse{}, fmt.Errorf("no killmail returned for killID=%d", killID)
	}

	// Cache it
	jsonBytes, err := json.Marshal(kills)
	if err == nil {
		zk.Cache.Set(cacheKey, jsonBytes, zkillCacheExpiration)
	}

	// Return the first (and typically only) kill
	return kills[0], nil
}

// doGetSingleKillMails is like doGetKillMails, but unmarshals into []model.ZkillMailFeedResponse
func (zk *zKillClient) doGetSingleKillMails(ctx context.Context, url string) ([]model.ZkillMailFeedResponse, error) {
	var kills []model.ZkillMailFeedResponse

	const maxAttempts = 5
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := zk.Client.Do(req)
		if err != nil {
			// HTTP request failed; sleep & retry
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		func() {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				// Decode the JSON
				if decodeErr := json.NewDecoder(resp.Body).Decode(&kills); decodeErr != nil {
					// If decode fails, we can log or handle the error
					// but we won't set 'kills' so we'll retry
				}
			case http.StatusTooManyRequests:
				// 429: handle backoff logic
				retryAfter := resp.Header.Get("Retry-After")
				if retryAfter != "" {
					if secs, errConv := strconv.Atoi(retryAfter); errConv == nil {
						time.Sleep(time.Duration(secs) * time.Second)
					} else {
						time.Sleep(backoff)
						backoff *= 2
					}
				} else {
					time.Sleep(backoff)
					backoff *= 2
				}
			default:
				// e.g. 404 or 500 - we can decide to retry or break
			}
		}()

		// If we successfully decoded kills, return immediately
		if len(kills) > 0 {
			return kills, nil
		}

		// If no kills, but status != 429, do exponential backoff & retry
		if resp.StatusCode != http.StatusTooManyRequests {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return nil, fmt.Errorf("all %d attempts failed for single kill URL %s", maxAttempts, url)
}
