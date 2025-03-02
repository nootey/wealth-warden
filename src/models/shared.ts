export interface GroupedItem {
    categoryName: string;
    total: number;
    months: Set<number>;
    spendingLimit: number|null;
    categoryType: string|null;
}

export interface Statistics {
    category: string;
    total: number;
    average: number;
    spending_limit: number|null;
    category_type: string|null;
}

export interface DynamicCategory {
    id: number|null;
    name: string;
}

export interface DynamicCategoryMapping {
    primary_links: object;
    primary_type: string;
    secondary_links: object;
    secondary_type: string;
}