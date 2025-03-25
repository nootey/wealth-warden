export interface SavingAllocation {
    id: number|null;
    inflow_category_id: number;
    inflow_category: object;
    amount: number;
    inflow_date: any;
    description: string|null;
}

export interface SavingsCategory {
    id: number|null;
    name: string;
}