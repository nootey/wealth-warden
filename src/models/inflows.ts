export interface Inflow {
    inflow_category_id: number;
    inflow_category: object;
    amount: number;
    inflow_date: any;
}

export interface InflowCategory {
    id: number;
    name: string;
}

export interface InflowStat {
    inflow_category_id: number;
    inflow_category_name: string;
    total_amount: number;
    month: number;
}

export interface InflowGroup {
    month: number;
    total_amount: number;
    inflow_category_id: number;
    inflow_category_name: string;
}