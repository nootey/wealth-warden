package ws

// Payloads carry identifiers only. Events are droppable, so a client must never
// need one to reach correct state.
const (
	TypeReportCompleted     = "report.completed"
	TypeReportFailed        = "report.failed"
	TypeNotificationCreated = "notification.created"
	TypeAssetPnLSynced      = "asset.pnl_synced"
	TypeUserJobUpdated      = "user_job.updated"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type ReportPayload struct {
	ReportID int64 `json:"report_id"`
}

type AssetPnLPayload struct {
	AssetID   *int64 `json:"asset_id,omitempty"`
	AccountID *int64 `json:"account_id,omitempty"`
}

type UserJobPayload struct {
	Kind  string `json:"kind"`
	State string `json:"state"`
}
