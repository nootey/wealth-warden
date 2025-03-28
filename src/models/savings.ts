export interface SavingAllocation {
    id: number|null;
    savings_category_id: number;
    savings_category: object;
    allocated_amount: number;
    savings_date: any;
}

export interface SavingsCategory {
    id: number|null;
    name: string;
    savings_type: string;
    goal_value: number;
    interest_rate: number|null;
    account_type: string;
}