import type {Account} from "./account_models.ts";

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
}

export interface Category {
    id: number | null;
    name: string;
    classification: string;
    parent_id: number | null;
}
