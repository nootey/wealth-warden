export interface MonthlyBudget {
    id: number|null;
    dynamic_category_id: number;
    dynamic_category: object;
    month: number;
    year: number;
    total_inflow: number;
    total_outflow: number;
    budget_inflow: number;
    budget_outflow: number;
    effective_budget: number;
    budget_snapshot: number;
    snapshot_threshold: number;
}

export interface MonthlyBudgetAllocation {
    id: number|null;
    monthly_budget_id: number;
    category: string;
    method: string;
    allocation: number|null;
    allocated_value: number;
    used_value: number|null;
}