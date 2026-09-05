import type { FilterObj } from "./shared_models.ts";

export const JOB_STATES = [
  "available",
  "running",
  "scheduled",
  "retryable",
  "pending",
  "discarded",
  "cancelled",
  "completed",
] as const;

export type JobState = (typeof JOB_STATES)[number];

export const JOB_KINDS = [
  "activity_log",
  "notification",
  "recalculate_asset_pnl",
  "backfill_asset_cash_flows",
  "sync_asset_after_trade",
  "recalculate_template_timezone",
  "correct_fee_accounting",
  "generate_category_report",
  "migrate_zero_cost_trades",
  "asset_history_backfill",
  "balance_backfill",
  "recurring_transactions",
  "asset_price_sync",
  "merge_categories",
  "merge_accounts",
] as const;

export interface RiverJob {
  id: number;
  kind: string;
  queue: string;
  state: JobState;
  attempt: number;
  max_attempts: number;
  priority: number;
  args: unknown;
  metadata: unknown;
  created_at: string;
  scheduled_at: string;
  attempted_at: string | null;
  finalized_at: string | null;
}

export interface RiverAttemptError {
  at: string;
  attempt: number;
  error: string;
  trace: string;
}

export interface RiverJobDetail extends RiverJob {
  attempted_by: string[] | null;
  tags: string[] | null;
  errors: RiverAttemptError[] | null;
}

export type JobCounts = Record<JobState, number>;

export interface RiverQueue {
  name: string;
  paused_at: string | null;
  counts: Record<string, number> | null;
}

export interface RiverPeriodicJob {
  id: string;
  kind: string;
  schedule: string;
  queue: string;
  last_run_at: string | null;
}

export interface JobListParams {
  rowsPerPage: number;
  sort: { field: string; order: number };
  states: JobState[];
  filters: FilterObj[];
}
