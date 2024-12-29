package zkill

import (
	"context"
	"github.com/guarzo/eveapi/common/model"
)

// ZKillService is a higher-level interface that uses ZKillClient to fetch multiple pages,
// aggregate data, etc.
type ZKillService interface {
	GetKillMailDataForMonth(ctx context.Context, params *model.Params, year, month int) ([]model.FlattenedKillMail, error)
	AggregateKillMailDumps(base, addition []model.FlattenedKillMail) []model.FlattenedKillMail
	AddEsiKillMail(ctx context.Context, mail model.ZkillMail, aggregatedData []model.FlattenedKillMail) ([]model.FlattenedKillMail, error)
	// etc. Add whatever else you need
}

// zKillService is the concrete struct implementing ZKillService.
type zKillService struct {
	ZKillClient
}

// NewZKillService constructs a zKillService using the given client & logger.
func NewZKillService(client ZKillClient) ZKillService {
	return &zKillService{
		ZKillClient: client,
	}
}

// Example method: fetch kills/losses for a given month, for the params’ corp/alli/characters
func (svc *zKillService) GetKillMailDataForMonth(ctx context.Context, params *model.Params, year, month int) ([]model.FlattenedKillMail, error) {
	var aggregated []model.FlattenedKillMail
	killMailIDs := make(map[int64]bool)

	entityGroups := map[string][]int{
		"corporation": params.Corporations,
		"alliance":    params.Alliances,
		"character":   params.Characters,
	}

	// example: we can fetch kills & losses up to some page limit
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

// processKillMails is an internal helper.
func (svc *zKillService) processKillMails(ctx context.Context, mails []model.ZkillMail, killMailIDs map[int64]bool, aggregated []model.FlattenedKillMail) ([]model.FlattenedKillMail, error) {
	for _, m := range mails {
		if _, exists := killMailIDs[m.KillMailID]; exists {
			// skip duplicates
			continue
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
func (svc *zKillService) AddEsiKillMail(ctx context.Context, mail model.ZkillMail, aggregated []model.FlattenedKillMail) ([]model.FlattenedKillMail, error) {
	// If you have an EsiService, you'd do:
	//   fullKill, err := svc.esiService.GetEsiKillMail(ctx, mail.KillMailID, mail.ZKB.Hash)
	//   if err != nil { return nil, err }

	// For now, let's just mock it or skip the ESI portion:
	// flatten them into a FlattenedKillMail
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
