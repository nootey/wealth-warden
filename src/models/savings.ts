export interface SavingAllocation {
    id: number|null;
    savings_category_id: number;
    savings_category: object;
    amount: number;
    savings_date: any;
    description: string|null;
}

export interface SavingsCategory {
    id: number|null;
    name: string;
    savings_type: string;
    goal_value: number;
    interest_rate: number|null;
    account_type: string;
}