export interface CategoryStat {
  category_id: number;
  category_name?: string | null;
  inflow: string;
  outflow: string;
  net: string;
  pct_of_inflow: number;
  pct_of_outflow: number;
}

export interface BasicAccountStats {
  user_id: number;
  account_id?: number | null;
  currency: string;
  year: number;
  inflow: string;
  outflow: string;
  net: string;
  avg_monthly_inflow: string;
  avg_monthly_outflow: string;
  take_home: string;
  overflow: string;
  avg_monthly_take_home: string;
  avg_monthly_overflow: string;
  active_months: number;
  categories?: CategoryStat[];
  generated_at: Date;
}

export interface MonthlyStats {
  user_id: number;
  account_id?: number | null;
  currency: string;
  year: number;
  inflow: string;
  outflow: string;
  net: string;
  take_home: string;
  overflow: string;
  savings: string;
  investments: string;
  debt_repayments: string;
  savings_rate: string;
  investments_rate: string;
  debt_repayment_rate: string;
  generated_at: Date;
  categories?: CategoryStat[];
}

export interface DailyStats {
  user_id: number;
  account_id?: number | null;
  currency: string;
  year: number;
  inflow: string;
  outflow: string;
  net: string;
  generated_at: Date;
}

export interface YearStatsWithAllocations {
  year: number;
  inflow: string;
  outflow: string;
  avg_monthly_inflow: string;
  avg_monthly_outflow: string;
  take_home: string;
  overflow: string;
  avg_monthly_take_home: string;
  avg_monthly_overflow: string;
  savings_allocated: string;
  investment_allocated: string;
  debt_allocated: string;
  total_allocated: string;
  savings_pct: number;
  investment_pct: number;
  debt_pct: number;
}

export interface YearlyBreakdownStats {
  current_year: YearStatsWithAllocations;
  comparison_year?: YearStatsWithAllocations | null;
}

export type ChartPoint = {
  date: string;
  value: number | string;
};

export type Change = {
  prev_period_end_date: string;
  prev_period_end_value: number;
  current_end_date: string;
  current_end_value: number;
  abs: number;
  pct: number;
};

export type NetworthResponse = {
  currency: string;
  points: ChartPoint[];
  current: ChartPoint;
  change?: Change;
  asset_type?: string;
};

export type YearlyCashFlowResponse = {
  year: number;
  months: MonthData[];
};

export type MonthData = {
  month: number;
  categories: {
    inflows: string | number;
    outflows: string | number;
    investments: string | number;
    savings: string | number;
    debt_repayments: string | number;
    take_home: string | number;
    overflow?: string | number;
  };
};

export interface YearStat {
  total: string;
  monthly_avg: string;
  months_with_data: number;
}

export interface YearlyCategoryStats {
  year_stats: Record<number, YearStat>;
  all_time_total: string;
  all_time_avg: string;
  all_time_months: number;
}

export interface CategoryFlow {
  category_id: number;
  category_name: string;
  amount: string;
  percentage: string;
}

export interface YearlySankeyData {
  year: number;
  currency: string;
  total_income: string;
  savings: string;
  investments: string;
  debt_repayments: string;
  expenses: string;
  expense_categories: CategoryFlow[];
}
