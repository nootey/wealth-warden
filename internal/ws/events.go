package ws

// Payloads carry identifiers only. Events are droppable, so a client must never
// need one to reach correct state.
const (
	TypeReportCompleted     = "report.completed"
	TypeReportFailed        = "report.failed"
	TypeNotificationCreated = "notification.created"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type ReportPayload struct {
	ReportID int64 `json:"report_id"`
}
