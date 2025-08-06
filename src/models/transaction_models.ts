export interface Transaction {
    id: number | null;
    account_id: number | null;
    category_id: number | null;
    transaction_type: string;
    amount: number | null;
    txn_date: Date | null;
    description: string | null;
}

export interface Category {
    id: number | null;
    name: string;
    classification: string;
    parent_id: number | null;
}
