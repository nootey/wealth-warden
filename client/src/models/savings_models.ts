export type SavingGoalStatus = "active" | "paused" | "completed" | "archived";
export type SavingContributionSource = "manual" | "auto";

export interface SavingGoal {
  id?: number;
  user_id: number;
  account_id: number;
  name: string;
  target_amount: string;
  current_amount: string;
  target_date?: string | null;
  status: SavingGoalStatus;
  priority: number;
  monthly_allocation?: string | null;
  fund_day_of_month?: number | null;
  created_at: string;
  updated_at?: string;
}

export interface SavingGoalWithProgress extends SavingGoal {
  progress_percent: string;
  track_status: "on_track" | "late" | "early" | "completed" | "no_target";
  months_remaining?: number;
  monthly_needed?: string;
}

export interface SavingContribution {
  id?: number;
  user_id: number;
  goal_id: number;
  amount: string;
  month: string;
  note?: string | null;
  source: SavingContributionSource;
  created_at: string;
  updated_at?: string;
}

export interface SavingGoalReq {
  account_id: number;
  name: string;
  target_amount: string;
  initial_amount?: string | null;
  target_date?: string | null;
  priority: number;
  monthly_allocation?: string | null;
  fund_day_of_month?: number | null;
}

export interface SavingGoalUpdateReq {
  name: string;
  target_amount: string;
  target_date?: string | null;
  status: SavingGoalStatus;
  priority: number;
  monthly_allocation?: string | null;
  fund_day_of_month?: number | null;
}

export interface SavingContributionReq {
  amount: string;
  month: string;
  note?: string | null;
}
