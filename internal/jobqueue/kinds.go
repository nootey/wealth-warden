package jobqueue

const (
	TypeActivityLog            = "activity_log"
	TypeRecalculateAssetPnL    = "recalculate_asset_pnl"
	TypeBackfillAssetCashFlows = "backfill_asset_cash_flows"
	TypeSyncAssetAfterTrade    = "sync_asset_after_trade"
	TypeRecalculateTemplateTZ  = "recalculate_template_timezone"
	TypeNotification           = "notification"
	TypeCorrectFeeAccounting   = "correct_fee_accounting"
	TypeGenerateCategoryReport = "generate_category_report"
	TypeMigrateZeroCostTrades  = "migrate_zero_cost_trades"
	TypeBackfillIncomeFXRates  = "backfill_income_fx_rates"
	TypeMergeCategories        = "merge_categories"
	TypeMergeAccounts          = "merge_accounts"
	TypeAssetHistoryBackfill   = "asset_history_backfill"
	TypeBalanceBackfill        = "balance_backfill"
	TypeRecurringTransactions  = "recurring_transactions"
	TypeAssetPriceSync         = "asset_price_sync"
)

// Kinds a user may see and act on from their own settings pages.
var SelfServiceKinds = map[string]bool{
	TypeMergeCategories: true,
	TypeMergeAccounts:   true,
}

// Two rebuilds at once would count every trade twice, so this queue runs one job
// at a time.
const QueueRebuild = "rebuild"

// Its own queue, so an 8 minute backfill cannot hold a worker slot a
// user-triggered job is waiting on.
const QueueScheduler = "scheduler"
