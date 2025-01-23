package zkill

import (
	"context"
	"github.com/guarzo/eveapi/common/model"
)

// ZKillService is a higher-level interface that uses ZKillClient to fetch multiple pages,
// aggregate data, single kills, etc.
type ZKillService interface {
	GetKillMailDataForMonth(ctx context.Context, params *model.Params, year, month int) ([]model.FlattenedKillMail, error)
	AggregateKillMailDumps(base, addition []model.FlattenedKillMail) []model.FlattenedKillMail
	AddEsiKillMail(ctx context.Context, mail model.ZkillMail, aggregated []model.FlattenedKillMail) ([]model.FlattenedKillMail, error)
	GetSingleKillmail(ctx context.Context, killID int) (model.ZkillMailFeedResponse, error)
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
