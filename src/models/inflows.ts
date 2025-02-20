export interface Inflow {
    inflow_category_id: number;
    inflow_category_name: string;
    total_amount: number;
    month: number;
}

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

export interface InflowGroup {
    month: number;
    total_amount: number;
    inflow_category_id: number;
    inflow_category_name: string;
}