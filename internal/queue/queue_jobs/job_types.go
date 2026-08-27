package queue_jobs

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
)

// Two rebuilds at once would count every trade twice, so this queue runs one job
// at a time.
const QueueRebuild = "rebuild"
