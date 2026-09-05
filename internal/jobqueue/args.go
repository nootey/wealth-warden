package jobqueue

import (
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/riverqueue/river"
)

type ActivityLogArgs struct {
	Event       string
	Category    string
	Description *string
	Payload     *utils.Changes
	Causer      *int64
}

func (ActivityLogArgs) Kind() string { return TypeActivityLog }

type NotificationArgs struct {
	Payload models.Notification
}

func (NotificationArgs) Kind() string { return TypeNotification }

type RecalculateAssetPnLArgs struct {
	UserID    int64
	AssetID   *int64 // nil = all assets for the account
	AccountID *int64 // nil = single asset mode
}

func (RecalculateAssetPnLArgs) Kind() string { return TypeRecalculateAssetPnL }

type SyncAssetAfterTradeArgs struct {
	UserID         int64
	AssetID        int64
	Ticker         string
	InvestmentType models.InvestmentType
	TradeDate      time.Time
}

func (SyncAssetAfterTradeArgs) Kind() string { return TypeSyncAssetAfterTrade }

type RecalculateTemplateTimezoneArgs struct {
	UserID      int64
	OldTimezone string
	NewTimezone string
}

func (RecalculateTemplateTimezoneArgs) Kind() string { return TypeRecalculateTemplateTZ }

type GenerateCategoryReportArgs struct {
	ReportID int64
	UserID   int64
	Params   models.CategoryReportParams
}

func (GenerateCategoryReportArgs) Kind() string { return TypeGenerateCategoryReport }

type BackfillAssetCashFlowsArgs struct{}

func (BackfillAssetCashFlowsArgs) Kind() string { return TypeBackfillAssetCashFlows }

func (BackfillAssetCashFlowsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

type CorrectFeeAccountingArgs struct{}

func (CorrectFeeAccountingArgs) Kind() string { return TypeCorrectFeeAccounting }

func (CorrectFeeAccountingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

type BackfillIncomeFXRatesArgs struct{}

func (BackfillIncomeFXRatesArgs) Kind() string { return TypeBackfillIncomeFXRates }

func (BackfillIncomeFXRatesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

// Internal* fields are worker-only and never rendered in the client.
type MergeCategoriesArgs struct {
	UserID                        int64
	InternalSourceCategoryID      int64
	InternalDestinationCategoryID int64
	SourceCategory                string
	DestinationCategory           string
}

func (MergeCategoriesArgs) Kind() string { return TypeMergeCategories }

// Internal* fields are worker-only and never rendered in the client.
type MergeAccountsArgs struct {
	UserID                       int64
	InternalSourceAccountID      int64
	InternalDestinationAccountID int64
	SourceAccount                string
	DestinationAccount           string
}

func (MergeAccountsArgs) Kind() string { return TypeMergeAccounts }

type MigrateZeroCostTradesArgs struct{}

func (MigrateZeroCostTradesArgs) Kind() string { return TypeMigrateZeroCostTrades }

// Shares the rebuild queue: the migration rewrites trades the rebuild jobs read.
func (MigrateZeroCostTradesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueRebuild}
}

type AssetPriceHistoryBackfillArgs struct{}

func (AssetPriceHistoryBackfillArgs) Kind() string { return TypeAssetHistoryBackfill }

type BalanceBackfillArgs struct{}

func (BalanceBackfillArgs) Kind() string { return TypeBalanceBackfill }

type RecurringTransactionsArgs struct{}

func (RecurringTransactionsArgs) Kind() string { return TypeRecurringTransactions }

type AssetPriceSyncArgs struct{}

func (AssetPriceSyncArgs) Kind() string { return TypeAssetPriceSync }
