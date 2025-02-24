export interface GroupedItem {
    categoryName: string;
    total: number;
    months: Set<number>;
}

export interface Statistics {
    category: string;
    total: number;
    average: number;
}