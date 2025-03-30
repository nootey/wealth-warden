export interface SavingAllocation {
    id: number|null;
    savings_category_id: number;
    savings_category: object;
    allocated_amount: number;
    allocation_date: any;
}
}

export interface SavingsCategory {
    id: number|null;
    name: string;
    savings_type: string;
    goal_target: number;
    interest_rate: number|null;
    account_type: string;
}

export interface SavingsGroup {
    month: number;
    total_amount: number;
    category_id: number;
    category_name: string;
    category_type: string;
}

export interface GroupedSavingsItem {
    categoryName: string;
    goalProgress: number | null;
    goalTarget: number | null;
    goalSpent: number | null;
}

export interface SavingsStatistics {
    category: string;
    average: number;
    category_type: string|null;
}