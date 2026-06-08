import type { Account } from "./account_models.ts";

export type InvestmentType = "stock" | "etf" | "crypto";

export type TradeType = "buy" | "sell";

export interface TickerData {
  name: string;
  exchange?: string;
  currency?: string;
}

export interface InvestmentAsset {
  id?: number | null;
  account?: Account | null;
  user_id?: number;
  investment_type: InvestmentType;
  name: string;
  ticker: string;
  quantity: string;
  average_buy_price?: string;
  value_at_buy?: string;
  current_value?: string;
  profit_loss?: string;
  profit_loss_percent?: string;
  current_price?: string | null;
  last_price_update?: Date | null;
  currency: string;
  created_at?: Date;
  updated_at?: Date;
  total_fees?: string;
  tax_summary?: AssetTaxSummary | null;
}

export type IncomeType = "staking_reward" | "dividend";

export interface InvestmentIncome {
  id?: number;
  asset_id: number;
  user_id?: number;
  txn_date: Date;
  income_type: IncomeType;
  quantity?: string | null;
  amount: string;
  tax_withheld?: string | null;
  currency: string;
  notes?: string | null;
  created_at?: Date;
  updated_at?: Date;
}

export interface InvestmentTrade {
  id?: number | null;
  user_id?: number;
  asset?: InvestmentAsset | null;
  txn_date: Date;
  trade_type: TradeType;
  quantity: string;
  fee: string;
  price_per_unit: string;
  value_at_buy?: string;
  current_value?: string;
  realized_value?: string;
  profit_loss?: string;
  profit_loss_percent?: string;
  currency: string;
  exchange_rate_to_usd?: string;
  description: string | null;
  created_at?: Date;
  updated_at?: Date;
  tax_info?: TradeTaxInfo | null;
}

export interface InvestmentTaxBracket {
  id: number;
  user_id: number;
  investment_type: InvestmentType;
  min_days_held: number;
  to_days: number | null;
  taxable_percent: string;
  label: string | null;
  created_at?: Date;
  updated_at?: Date;
}

export interface InvestmentTaxSettings {
  id?: number;
  user_id?: number;
  loss_offsetting_enabled: boolean;
}

export interface TradeTaxInfo {
  days_held: number;
  taxable_percent: string | null;
  taxable_profit: string;
  days_until_next_bracket: number | null;
  days_until_tax_free: number | null;
}

export interface AssetTaxSummary {
  estimated_tax_due: string;
  after_tax_pnl: string;
}
