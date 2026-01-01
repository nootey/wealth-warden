import type { Account } from "./account_models.ts";

export interface Transaction {
  id: number | null;
  account_id: number | null;
  category_id: number | null;
  category: Category | null;
  transaction_type: string;
  amount: string | null;
  txn_date: Date | null;
  description: string | null;
  account: Account;
  deleted_at: Date | null;
  created_at?: Date;
  updated_at?: Date;
  is_adjustment: boolean;
}

export interface Transfer {
  source_id: number | null;
  destination_id: number | null;
  amount: string | null;
  notes: string | null;
  deleted_at: Date | null;
  created_at?: Date | null;
  from: Transaction | null;
  to: Transaction | null;
}

export interface Category {
  id?: number | null;
  name: string;
  display_name: string;
  classification: string;
  parent_id: number | null;
  is_default: boolean;
  deleted_at: Date | null;
}

export interface CategoryGroup {
  id?: number | null;
  name: string;
  classification: string;
  description: string | null;
  categories?: Category[];
}

export interface CategoryOrGroup {
  id: number;
  name: string;
  is_group: boolean;
  classification: string;
  category_ids: number[];
}

export interface TransactionTemplate {
  id: number | null;
  name: string;
  account_id: number | null;
  category_id: number | null;
  transaction_type: string;
  amount: string | null;
  period: string;
  frequency: string;
  next_run_at: Date | null;
  last_run_at?: Date | null;
  run_count?: number;
  end_date?: Date | null;
  max_runs?: number | null;
  is_active: boolean;
  category: Category | null;
  account: Account;
}
