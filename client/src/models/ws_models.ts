export type WsEventType =
  | "report.completed"
  | "report.failed"
  | "notification.created"
  | "asset.pnl_synced"
  | "user_job.updated";

export interface WsEvent {
  type: WsEventType;
  payload?: unknown;
}

export interface ReportPayload {
  report_id: number;
}

export interface AssetPnLPayload {
  asset_id?: number;
  account_id?: number;
}

export interface UserJobPayload {
  kind: string;
  state: string;
}

export type WsHandler = (payload: unknown) => void;
