export interface AccountType {
  id: number | null;
  name: string;
  type: string;
  sub_type: string;
  classification: string;
}

export interface AccountBalance {
  balance: string | null;
  market_value: string | null;
  total_balance: string | null;
}

export interface Account {
  id: number | null;
  name: string;
  account_type: AccountType;
  balance: AccountBalance;
  currency?: string;
  is_active: boolean;
  expected_balance?: string;
  balance_projection?: string;
  opened_at?: Date | null;
  closed_at: Date | null;
  is_default?: boolean;
  credit_limit?: string | null;
}

export interface AccountWithOpening extends Account {
  start_balance: string | null;
}

export interface AccountLookup {
  id: number;
  name: string;
  currency: string;
  closed_at: string | null;
}
