export type ChartPoint = {
    date: string;
    value: number | string
}

export type Change = {
    prev_period_end_date: string;
    prev_period_end_value: number;
    current_end_date: string;
    current_end_value: number;
    abs: number;
    pct: number;
};


export type NetworthResponse = {
    currency: string;
    points: ChartPoint[];
    current: ChartPoint;
    change?: Change;
    asset_type?: string
};

export type MonthlyCashFlow = {
    month: number;
    inflows: string[];
    outflows: string[];
    net: string;
}

export type MonthlyCashFlowResponse = {
    year: number;
    series: MonthlyCashFlow[];
}

export interface MonthlyCategoryUsage {
    month: number;
    category_id: number;
    category: string;
    amount: string;
    percentage?: string;
}