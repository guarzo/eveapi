package zkill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/guarzo/eveapi/common/model"
)

// ZKillService is a higher-level interface that uses ZKillClient to fetch multiple pages,
// aggregate data, single kills, etc.
type ZKillService interface {
	GetKillMailDataForMonth(ctx context.Context, params *model.Params, year, month int) ([]model.FlattenedKillMail, error)
	AggregateKillMailDumps(base, addition []model.FlattenedKillMail) []model.FlattenedKillMail
	AddEsiKillMail(ctx context.Context, mail model.ZkillMail, aggregated []model.FlattenedKillMail) ([]model.FlattenedKillMail, error)
}

// zKillService is the concrete struct implementing ZKillService.
type zKillService struct {
	ZKillClient
}

// NewZKillService constructs a zKillService using the given client.
func NewZKillService(client ZKillClient) ZKillService {
	return &zKillService{
		ZKillClient: client,
	}
}

// GetKillMailDataForMonth is an example method: fetch kills/losses for a given month.
func (svc *zKillService) GetKillMailDataForMonth(
	ctx context.Context,
	params *model.Params,
	year, month int,
) ([]model.FlattenedKillMail, error) {

	var aggregated []model.FlattenedKillMail
	killMailIDs := make(map[int64]bool)

	entityGroups := map[string][]int{
		"corporation": params.Corporations,
		"alliance":    params.Alliances,
		"character":   params.Characters,
	}

	const maxPages = 100
	for etype, ids := range entityGroups {
		for _, id := range ids {
			// 1) Kills
			for page := 1; page <= maxPages; page++ {
				kills, err := svc.ZKillClient.GetKillsPageData(ctx, etype, id, page, year, month)
				if err != nil {
					break
				}
				if len(kills) == 0 {
					break
				}
				updated, err := svc.processKillMails(ctx, kills, killMailIDs, aggregated)
				if err != nil {
					break
				}
				aggregated = updated
			}

			// 2) Losses
			for page := 1; page <= maxPages; page++ {
				losses, err := svc.ZKillClient.GetLossPageData(ctx, etype, id, page, year, month)
				if err != nil {
					break
				}
				if len(losses) == 0 {
					break
				}
				updated, err := svc.processKillMails(ctx, losses, killMailIDs, aggregated)
				if err != nil {
					break
				}
				aggregated = updated
			}
		}
	}

	return aggregated, nil
}

// processKillMails is an internal helper to flatten & deduplicate killmails.
func (svc *zKillService) processKillMails(
	ctx context.Context,
	mails []model.ZkillMail,
	killMailIDs map[int64]bool,
	aggregated []model.FlattenedKillMail,
) ([]model.FlattenedKillMail, error) {

	for _, m := range mails {
		if _, exists := killMailIDs[m.KillMailID]; exists {
			continue // skip duplicates
		}
		updated, err := svc.AddEsiKillMail(ctx, m, aggregated)
		if err != nil {
			continue
		}
		aggregated = updated
		killMailIDs[m.KillMailID] = true
	}
	return aggregated, nil
}

// AggregateKillMailDumps merges two slices of FlattenedKillMail
func (svc *zKillService) AggregateKillMailDumps(base, addition []model.FlattenedKillMail) []model.FlattenedKillMail {
	if base == nil {
		return addition
	}
	if addition == nil {
		return base
	}
	return append(base, addition...)
}

// AddEsiKillMail is a stub showing how you might fetch from ESI, then flatten it.
func (svc *zKillService) AddEsiKillMail(
	ctx context.Context,
	mail model.ZkillMail,
	aggregated []model.FlattenedKillMail,
) ([]model.FlattenedKillMail, error) {
	// Example: If you have an EsiService, you'd do:
	//   fullKill, err := svc.esiService.GetEsiKillMail(ctx, mail.KillMailID, mail.ZKB.Hash)
	//   if err != nil { return aggregated, err }
	//   flatten it -> FlattenedKillMail

	flattened := model.FlattenedKillMail{
		KillMailID:   mail.KillMailID,
		Hash:         mail.ZKB.Hash,
		TotalValue:   mail.ZKB.TotalValue,
		DroppedValue: mail.ZKB.DroppedValue,
		// etc.
	}
	aggregated = append(aggregated, flattened)
	return aggregated, nil
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
