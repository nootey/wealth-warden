package jobqueue_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

func payloadKeys(t *testing.T, args jobqueue.Job) []string {
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

func assertKeys(t *testing.T, args jobqueue.Job, want ...string) {
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

	assertKeys(t, jobqueue.ActivityLogArgs{
		Event:       "account.update",
		Category:    "account",
		Description: &desc,
		Payload:     utils.InitChanges(),
		Causer:      &causer,
	}, "Event", "Category", "Description", "Payload", "Causer")

	assertKeys(t, jobqueue.RecalculateAssetPnLArgs{
		UserID:    1,
		AssetID:   &assetID,
		AccountID: &accountID,
	}, "UserID", "AssetID", "AccountID")

	assertKeys(t, jobqueue.SyncAssetAfterTradeArgs{
		UserID:         1,
		AssetID:        12,
		Ticker:         "AAPL",
		InvestmentType: models.InvestmentStock,
		TradeDate:      time.Now(),
	}, "UserID", "AssetID", "Ticker", "InvestmentType", "TradeDate")

	assertKeys(t, jobqueue.RecalculateTemplateTimezoneArgs{
		UserID:      1,
		OldTimezone: "Europe/Paris",
		NewTimezone: "America/New_York",
	}, "UserID", "OldTimezone", "NewTimezone")

	assertKeys(t, jobqueue.NotificationArgs{
		Payload: models.Notification{UserID: 1, Title: "hi"},
	}, "Payload")

	assertKeys(t, jobqueue.GenerateCategoryReportArgs{
		ReportID: 9,
		UserID:   1,
		Params:   models.CategoryReportParams{Years: []int{2026}, Description: "d"},
	}, "ReportID", "UserID", "Params")

	assertKeys(t, jobqueue.BackfillAssetCashFlowsArgs{})
	assertKeys(t, jobqueue.CorrectFeeAccountingArgs{})
}

func TestPayloadRoundTrip(t *testing.T) {
	assetID := int64(12)
	orig := jobqueue.GenerateCategoryReportArgs{
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
	var got jobqueue.GenerateCategoryReportArgs
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, orig)
	}

	sync := jobqueue.SyncAssetAfterTradeArgs{UserID: 1, AssetID: assetID, Ticker: "AAPL", InvestmentType: models.InvestmentStock, TradeDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)}
	raw, _ = json.Marshal(sync)
	var gotSync jobqueue.SyncAssetAfterTradeArgs
	if err := json.Unmarshal(raw, &gotSync); err != nil {
		t.Fatalf("unmarshal sync: %v", err)
	}
	if !gotSync.TradeDate.Equal(sync.TradeDate) || gotSync.Ticker != sync.Ticker {
		t.Errorf("sync round-trip mismatch: got %+v, want %+v", gotSync, sync)
	}
}

// Kinds are persisted on river_job rows; a rename orphans in-flight jobqueue.
func TestJobKinds(t *testing.T) {
	// Slice, not map: args holding slices are not hashable.
	cases := []struct {
		args jobqueue.Job
		want string
	}{
		{jobqueue.ActivityLogArgs{}, jobqueue.TypeActivityLog},
		{jobqueue.RecalculateAssetPnLArgs{}, jobqueue.TypeRecalculateAssetPnL},
		{jobqueue.BackfillAssetCashFlowsArgs{}, jobqueue.TypeBackfillAssetCashFlows},
		{jobqueue.SyncAssetAfterTradeArgs{}, jobqueue.TypeSyncAssetAfterTrade},
		{jobqueue.RecalculateTemplateTimezoneArgs{}, jobqueue.TypeRecalculateTemplateTZ},
		{jobqueue.NotificationArgs{}, jobqueue.TypeNotification},
		{jobqueue.CorrectFeeAccountingArgs{}, jobqueue.TypeCorrectFeeAccounting},
		{jobqueue.GenerateCategoryReportArgs{}, jobqueue.TypeGenerateCategoryReport},
		{jobqueue.MigrateZeroCostTradesArgs{}, jobqueue.TypeMigrateZeroCostTrades},
		{jobqueue.AssetPriceHistoryBackfillArgs{}, jobqueue.TypeAssetHistoryBackfill},
		{jobqueue.BalanceBackfillArgs{}, jobqueue.TypeBalanceBackfill},
		{jobqueue.RecurringTransactionsArgs{}, jobqueue.TypeRecurringTransactions},
		{jobqueue.AssetPriceSyncArgs{}, jobqueue.TypeAssetPriceSync},
	}
	for _, tc := range cases {
		if got := tc.args.Kind(); got != tc.want {
			t.Errorf("Kind() = %q, want %q", got, tc.want)
		}
	}
}
