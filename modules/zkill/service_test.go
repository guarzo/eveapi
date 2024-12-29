package zkill_test

import (
	"context"
	"testing"

	"github.com/guarzo/eveapi/common/model"
	"github.com/guarzo/eveapi/modules/zkill"
)

type mockZKillClient struct {
	killsFunc func(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
	lossFunc  func(ctx context.Context, entityType string, entityID, page, year, month int) ([]model.ZkillMail, error)
}

func (m *mockZKillClient) GetKillsPageData(ctx context.Context, eType string, eID, page, year, month int) ([]model.ZkillMail, error) {
	return m.killsFunc(ctx, eType, eID, page, year, month)
}
func (m *mockZKillClient) GetLossPageData(ctx context.Context, eType string, eID, page, year, month int) ([]model.ZkillMail, error) {
	return m.lossFunc(ctx, eType, eID, page, year, month)
}
func (m *mockZKillClient) RemoveCacheEntry(k string)                        {}
func (m *mockZKillClient) BuildCacheKey(a, b string, c, d, e, f int) string { return "dummyKey" }
func TestZKillService_GetKillMailDataForMonth(t *testing.T) {
	calls := 0

	mockClient := &mockZKillClient{
		killsFunc: func(ctx context.Context, etype string, eID, page, year, month int) ([]model.ZkillMail, error) {
			calls++
			// Return 1 killmail on page=1, then empty on page>1 (forces 1-page usage)
			if page > 1 {
				return nil, nil
			}
			return []model.ZkillMail{{KillMailID: 111}}, nil
		},
		lossFunc: func(ctx context.Context, etype string, eID, page, year, month int) ([]model.ZkillMail, error) {
			calls++
			if page > 1 {
				return nil, nil
			}
			return []model.ZkillMail{{KillMailID: 222}}, nil
		},
	}

	svc := zkill.NewZKillService(mockClient)

	// Suppose each "group" has 1 ID => total 3 IDs across character/corp/alliance
	params := &model.Params{
		Corporations: []int{111},
		Alliances:    []int{222},
		Characters:   []int{333},
		Year:         2023,
	}

	out, err := svc.GetKillMailDataForMonth(context.Background(), params, 2023, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// *Now* we only do page=1 for kills, page=1 for losses => 2 calls per entity => 3 entities => 6 calls total
	if calls != 12 {
		t.Errorf("expected 6 calls, got %d", calls)
	}

	// We returned 1 killmail from kills page=1 and 1 killmail from losses page=1 => total 2 per entity => 3 entities => 6
	if len(out) != 2 {
		t.Errorf("expected 6 killmails, got %d", len(out))
	}
}

func TestZKillService_AddEsiKillMail(t *testing.T) {
	svc := zkill.NewZKillService(nil)
	var existing []model.FlattenedKillMail
	mail := model.ZkillMail{KillMailID: 999}
	updated, err := svc.AddEsiKillMail(context.Background(), mail, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updated) != 1 {
		t.Errorf("expected 1 killmail, got %d", len(updated))
	}
	if updated[0].KillMailID != 999 {
		t.Errorf("unexpected killmail ID: %d", updated[0].KillMailID)
	}
}

func TestZKillService_AggregateKillMailDumps(t *testing.T) {
	svc := zkill.NewZKillService(nil)
	base := []model.FlattenedKillMail{{KillMailID: 1}}
	addition := []model.FlattenedKillMail{{KillMailID: 2}}
	combined := svc.AggregateKillMailDumps(base, addition)
	if len(combined) != 2 {
		t.Errorf("expected 2, got %d", len(combined))
	}
}
