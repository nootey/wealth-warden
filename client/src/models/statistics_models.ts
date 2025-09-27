export interface CategoryStat {
    category_id: number;
    category_name?: string | null;
    inflow: string;
    outflow: string;
    net: string;
    pct_of_inflow: number;
    pct_of_outflow: number;
}

export interface BasicAccountStats {
    user_id: number;
    account_id?: number | null;
    currency: string;
    year: number;
    inflow: string;
    outflow: string;
    net: string;
    avg_monthly_inflow: string;
    avg_monthly_outflow: string;
    take_home: string;
    overflow: string;
    avg_monthly_take_home: string;
    avg_monthly_overflow: string;
    active_months: number;
    categories?: CategoryStat[];
    generated_at: Date;
}