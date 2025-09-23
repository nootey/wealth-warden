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
};