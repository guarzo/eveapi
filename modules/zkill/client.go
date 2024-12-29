package zkill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
)

// ZKillClient is a lower-level interface for fetching from zKillboard’s API.
type ZKillClient interface {
	GetKillsPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
	GetLossPageData(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
	RemoveCacheEntry(cacheKey string)
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
