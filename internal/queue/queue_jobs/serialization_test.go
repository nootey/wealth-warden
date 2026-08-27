package queue_jobs_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/queue/queue_jobs"
	"wealth-warden/pkg/utils"
)

func payloadKeys(t *testing.T, args queue.Job) []string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal %s: %v", args.Kind(), err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode payload to map: %v", err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func assertKeys(t *testing.T, args queue.Job, want ...string) {
	t.Helper()
	got := payloadKeys(t, args)
	sort.Strings(want)
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s payload keys = %v, want %v", args.Kind(), got, want)
	}
}

// Args carry data fields only: a dep on an args struct adds a key here.
func TestPayloadContract(t *testing.T) {
	desc := "moved"
	causer := int64(7)
	assetID := int64(12)
	accountID := int64(3)

	assertKeys(t, queue_jobs.ActivityLogArgs{
		Event:       "account.update",
		Category:    "account",
		Description: &desc,
		Payload:     utils.InitChanges(),
		Causer:      &causer,
	}, "Event", "Category", "Description", "Payload", "Causer")

	assertKeys(t, queue_jobs.RecalculateAssetPnLArgs{
		UserID:    1,
		AssetID:   &assetID,
		AccountID: &accountID,
	}, "UserID", "AssetID", "AccountID")

	assertKeys(t, queue_jobs.SyncAssetAfterTradeArgs{
		UserID:         1,
		AssetID:        12,
		Ticker:         "AAPL",
		InvestmentType: models.InvestmentStock,
		TradeDate:      time.Now(),
	}, "UserID", "AssetID", "Ticker", "InvestmentType", "TradeDate")

	assertKeys(t, queue_jobs.RecalculateTemplateTimezoneArgs{
		UserID:      1,
		OldTimezone: "Europe/Paris",
		NewTimezone: "America/New_York",
	}, "UserID", "OldTimezone", "NewTimezone")

	assertKeys(t, queue_jobs.NotificationArgs{
		Payload: models.Notification{UserID: 1, Title: "hi"},
	}, "Payload")

	assertKeys(t, queue_jobs.GenerateCategoryReportArgs{
		ReportID: 9,
		UserID:   1,
		Params:   models.CategoryReportParams{Years: []int{2026}, Description: "d"},
	}, "ReportID", "UserID", "Params")

	assertKeys(t, queue_jobs.BackfillAssetCashFlowsArgs{})
	assertKeys(t, queue_jobs.CorrectFeeAccountingArgs{})
}

func TestPayloadRoundTrip(t *testing.T) {
	assetID := int64(12)
	orig := queue_jobs.GenerateCategoryReportArgs{
		ReportID: 9,
		UserID:   1,
		Params: models.CategoryReportParams{
			InflowCategoryIDs:  []int64{1, 2},
			OutflowCategoryIDs: []int64{3},
			Years:              []int{2025, 2026},
			Description:        "rent",
			AllTime:            true,
		},
	}
	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got queue_jobs.GenerateCategoryReportArgs
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, orig)
	}

	sync := queue_jobs.SyncAssetAfterTradeArgs{UserID: 1, AssetID: assetID, Ticker: "AAPL", InvestmentType: models.InvestmentStock, TradeDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)}
	raw, _ = json.Marshal(sync)
	var gotSync queue_jobs.SyncAssetAfterTradeArgs
	if err := json.Unmarshal(raw, &gotSync); err != nil {
		t.Fatalf("unmarshal sync: %v", err)
	}
	if !gotSync.TradeDate.Equal(sync.TradeDate) || gotSync.Ticker != sync.Ticker {
		t.Errorf("sync round-trip mismatch: got %+v, want %+v", gotSync, sync)
	}
}

// Kinds are persisted on river_job rows; a rename orphans in-flight jobs.
func TestJobKinds(t *testing.T) {
	// Slice, not map: args holding slices are not hashable.
	cases := []struct {
		args queue.Job
		want string
	}{
		{queue_jobs.ActivityLogArgs{}, queue_jobs.TypeActivityLog},
		{queue_jobs.RecalculateAssetPnLArgs{}, queue_jobs.TypeRecalculateAssetPnL},
		{queue_jobs.BackfillAssetCashFlowsArgs{}, queue_jobs.TypeBackfillAssetCashFlows},
		{queue_jobs.SyncAssetAfterTradeArgs{}, queue_jobs.TypeSyncAssetAfterTrade},
		{queue_jobs.RecalculateTemplateTimezoneArgs{}, queue_jobs.TypeRecalculateTemplateTZ},
		{queue_jobs.NotificationArgs{}, queue_jobs.TypeNotification},
		{queue_jobs.CorrectFeeAccountingArgs{}, queue_jobs.TypeCorrectFeeAccounting},
		{queue_jobs.GenerateCategoryReportArgs{}, queue_jobs.TypeGenerateCategoryReport},
	}
	for _, tc := range cases {
		if got := tc.args.Kind(); got != tc.want {
			t.Errorf("Kind() = %q, want %q", got, tc.want)
		}
	}
}
