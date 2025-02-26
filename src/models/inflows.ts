export interface Inflow {
    id: number|null;
    inflow_category_id: number;
    inflow_category: object;
    amount: number;
    inflow_date: any;
    description: string|null;
}

export interface InflowCategory {
    id: number|null;
    name: string;
}

export interface InflowStat {
    category_id: number;
    category_name: string;
    total_amount: number;
    month: number;
}

export interface InflowGroup {
    month: number;
    total_amount: number;
    inflow_category_id: number;
    inflow_category_name: string;
}