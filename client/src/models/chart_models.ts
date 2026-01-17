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

export type MonthlyCashFlow = {
  month: number;
  inflows: string[];
  outflows: string[];
  net: string;
};

export type MonthlyCashFlowResponse = {
  year: number;
  series: MonthlyCashFlow[];
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
