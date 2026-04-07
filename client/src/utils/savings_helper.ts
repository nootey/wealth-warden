import type {
  SavingGoalStatus,
  SavingGoalWithProgress,
} from "../models/savings_models.ts";

const STATUS_SORT_ORDER: Record<SavingGoalStatus, number> = {
  active: 0,
  paused: 1,
  completed: 2,
  archived: 3,
};

const savingsHelper = {
  trackStatusLabel(status: string): string {
    const map: Record<string, string> = {
      on_track: "On track",
      early: "Ahead",
      late: "Behind",
      completed: "Completed",
      no_target: "No target",
    };
    return map[status] ?? status;
  },
  trackStatusSeverity(status: string): string {
    const map: Record<string, string> = {
      on_track: "success",
      early: "info",
      late: "warn",
      completed: "success",
      no_target: "secondary",
    };
    return map[status] ?? "secondary";
  },
  goalStatusLabel(status: SavingGoalStatus): string {
    const map: Record<SavingGoalStatus, string> = {
      active: "Active",
      paused: "Paused",
      completed: "Completed",
      archived: "Archived",
    };
    return map[status] ?? status;
  },
  goalStatusSeverity(status: SavingGoalStatus): string {
    const map: Record<SavingGoalStatus, string> = {
      active: "success",
      paused: "warn",
      completed: "success",
      archived: "secondary",
    };
    return map[status] ?? "secondary";
  },
  goalSortOrder(status: SavingGoalStatus): number {
    return STATUS_SORT_ORDER[status] ?? 99;
  },
  isGoalDimmed(status: SavingGoalStatus): boolean {
    return status === "paused" || status === "archived";
  },
  progressPercent(goal: SavingGoalWithProgress): number {
    const p = Number(goal.progress_percent);
    return isNaN(p) ? 0 : parseFloat(Math.min(p, 100).toFixed(2));
  },
};

export default savingsHelper;
