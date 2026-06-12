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

// payloadKeys marshals a job the way DBDispatcher does and returns the top-level
// JSON keys. Dependency fields must never appear here.
func payloadKeys(t *testing.T, job queue.Job) []string {
	t.Helper()
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal %s: %v", job.Type(), err)
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

func assertKeys(t *testing.T, job queue.Job, want ...string) {
	t.Helper()
	got := payloadKeys(t, job)
	sort.Strings(want)
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s payload keys = %v, want %v", job.Type(), got, want)
	}
}

// TestPayloadContract is the core invariant of the durable queue: serialized
// payloads carry data fields only — never the live deps that jobs embed. A new
// dep without `json:"-"` adds a key here; an unexported data field drops one.
func TestPayloadContract(t *testing.T) {
	desc := "moved"
	causer := int64(7)
	assetID := int64(12)
	accountID := int64(3)

	assertKeys(t, &queue_jobs.ActivityLogJob{
		Event:       "account.update",
		Category:    "account",
		Description: &desc,
		Payload:     utils.InitChanges(),
		Causer:      &causer,
	}, "Event", "Category", "Description", "Payload", "Causer")

	assertKeys(t, &queue_jobs.RecalculateAssetPnLJob{
		UserID:    1,
		AssetID:   &assetID,
		AccountID: &accountID,
	}, "UserID", "AssetID", "AccountID")

	assertKeys(t, &queue_jobs.SyncAssetAfterTradeJob{
		UserID:         1,
		AssetID:        12,
		Ticker:         "AAPL",
		InvestmentType: models.InvestmentStock,
		TradeDate:      time.Now(),
	}, "UserID", "AssetID", "Ticker", "InvestmentType", "TradeDate")

	assertKeys(t, &queue_jobs.RecalculateTemplateTimezoneJob{
		UserID:      1,
		OldTimezone: "Europe/Paris",
		NewTimezone: "America/New_York",
	}, "UserID", "OldTimezone", "NewTimezone")

	assertKeys(t, &queue_jobs.NotificationJob{
		Payload: models.Notification{UserID: 1, Title: "hi"},
	}, "Payload")

	assertKeys(t, &queue_jobs.GenerateCategoryReportJob{
		ReportID: 9,
		UserID:   1,
		Params:   models.CategoryReportParams{Years: []int{2026}, Description: "d"},
	}, "ReportID", "UserID", "Params")

	// Payload-less maintenance jobs serialize to an empty object — deps dropped.
	assertKeys(t, &queue_jobs.BackfillAssetCashFlowsJob{})
	assertKeys(t, &queue_jobs.CorrectFeeAccountingJob{})
}

// TestPayloadRoundTrip confirms data survives marshal → unmarshal unchanged, the
// path the consumer's registry relies on to rebuild jobs.
func TestPayloadRoundTrip(t *testing.T) {
	assetID := int64(12)
	orig := &queue_jobs.GenerateCategoryReportJob{
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
	var got queue_jobs.GenerateCategoryReportJob
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(orig.Params, got.Params) || got.ReportID != orig.ReportID || got.UserID != orig.UserID {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, *orig)
	}

	sync := &queue_jobs.SyncAssetAfterTradeJob{UserID: 1, AssetID: assetID, Ticker: "AAPL", InvestmentType: models.InvestmentStock, TradeDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)}
	raw, _ = json.Marshal(sync)
	var gotSync queue_jobs.SyncAssetAfterTradeJob
	if err := json.Unmarshal(raw, &gotSync); err != nil {
		t.Fatalf("unmarshal sync: %v", err)
	}
	if !gotSync.TradeDate.Equal(sync.TradeDate) || gotSync.Ticker != sync.Ticker {
		t.Errorf("sync round-trip mismatch: got %+v, want %+v", gotSync, *sync)
	}
}

// TestJobTypeTags guards the stable type tags persisted on rows — a rename here
// would orphan in-flight jobs.
func TestJobTypeTags(t *testing.T) {
	cases := map[queue.Job]string{
		&queue_jobs.ActivityLogJob{}:                 queue_jobs.TypeActivityLog,
		&queue_jobs.RecalculateAssetPnLJob{}:         queue_jobs.TypeRecalculateAssetPnL,
		&queue_jobs.BackfillAssetCashFlowsJob{}:      queue_jobs.TypeBackfillAssetCashFlows,
		&queue_jobs.SyncAssetAfterTradeJob{}:         queue_jobs.TypeSyncAssetAfterTrade,
		&queue_jobs.RecalculateTemplateTimezoneJob{}: queue_jobs.TypeRecalculateTemplateTZ,
		&queue_jobs.NotificationJob{}:                queue_jobs.TypeNotification,
		&queue_jobs.CorrectFeeAccountingJob{}:        queue_jobs.TypeCorrectFeeAccounting,
		&queue_jobs.GenerateCategoryReportJob{}:      queue_jobs.TypeGenerateCategoryReport,
	}
	for job, want := range cases {
		if got := job.Type(); got != want {
			t.Errorf("Type() = %q, want %q", got, want)
		}
	}
}
