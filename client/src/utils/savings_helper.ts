import type { SavingGoalWithProgress } from "../models/savings_models.ts";

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
  progressPercent(goal: SavingGoalWithProgress): number {
    const p = Number(goal.progress_percent);
    return isNaN(p) ? 0 : Math.min(p, 100);
  },
};

export default savingsHelper;
