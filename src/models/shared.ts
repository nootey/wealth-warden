export interface GroupedItem {
    categoryName: string;
    total: number;
    months: Set<number>;
    spendingLimit: number|null;
}

export interface Statistics {
    category: string;
    total: number;
    average: number;
    spending_limit: number|null;
}