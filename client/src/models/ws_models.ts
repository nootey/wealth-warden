export type WsEventType =
  "report.completed" | "report.failed" | "notification.created";

export interface WsEvent {
  type: WsEventType;
  payload?: unknown;
}

export interface ReportPayload {
  report_id: number;
}

export type WsHandler = (payload: unknown) => void;
