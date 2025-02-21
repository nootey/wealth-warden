export interface Inflow {
    inflow_category_id: number;
    inflow_category: object;
    amount: number;
    inflow_date: any;
}

export interface ReoccurringInflow {
    startDate: number;
    endDate: string;
    intervalValue: number;
    intervalUnit: number;
}

export interface InflowCategory {
    id: number;
    name: string;
}

export interface GroupedItem {
    categoryName: string;
    total: number;
    months: Set<number>;
}

export interface InflowStat {
    inflow_category_id: number;
    inflow_category_name: string;
    total_amount: number;
    month: number;
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