export interface MonthlyBudget {
    id: number|null;
    dynamic_category_id: number;
    dynamic_category: object;
    month: number;
    year: number;
    total_inflow: number;
    total_outflow: number;
    effective_budget: number;
    budget_snapshot: number;
}