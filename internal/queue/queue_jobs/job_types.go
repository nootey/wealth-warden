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
)
