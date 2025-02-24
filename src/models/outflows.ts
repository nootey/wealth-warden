export interface Outflow {
    outflow_category_id: number;
    outflow_category: object;
    amount: number;
    outflow_date: any;
}

export interface OutflowCategory {
    id: number;
    name: string;
    spending_limit: number;
    outflow_type: string;
}

export interface OutflowStat {
    outflow_category_id: number;
    outflow_category_name: string;
    total_amount: number;
    month: number;
}

export interface OutflowGroup {
    month: number;
    total_amount: number;
    outflow_category_id: number;
    outflow_category_name: string;
}