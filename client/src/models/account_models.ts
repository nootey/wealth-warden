export interface AccountType {
  id: number | null;
  name: string;
  type: string;
  sub_type: string;
  classification: string;
}

export interface Balance {
  id: number | null;
  as_of: Date | null;
  start_balance: string | null;
  end_balance: string | null;
}

export interface Account {
  id: number | null;
  name: string;
  account_type: AccountType;
  balance: Balance;
  currency?: string;
  is_active: boolean;
  expected_balance?: string;
  balance_projection?: string;
  opened_at?: Date | null;
  closed_at: Date | null;
  is_default?: boolean;
}
