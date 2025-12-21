export type InvestmentType = 'stock' | 'etf' | 'crypto';

export type TransactionType = 'buy' | 'sell';

export interface InvestmentHolding {
    id?: number | null;
    account_id: number | null;
    user_id?: number;
    investment_type: InvestmentType;
    name: string;
    ticker: string;
    quantity: string;
    average_buy_price?: string;
    current_price?: string | null;
    last_price_update?: Date | null;
    created_at?: Date;
    updated_at?: Date;
}

export interface InvestmentTransaction {
    id: number | null;
    account_id: number;
    user_id: number;
    holding_id: number;
    transaction_type: TransactionType;
    name: string;
    ticker: string;
    quantity: string;
    fee: string;
    price_per_unit: string;
    value_at_buy: string;
    currency: string;
    exchange_rate_to_usd: string;
    txn_date: Date;
    created_at: Date;
    updated_at: Date;
}