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
}
