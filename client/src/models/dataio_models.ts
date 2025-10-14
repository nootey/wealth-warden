type TxnSample = {
    transaction_type: string;
    amount: string;
    currency: string;
    category: string;
    description: string;
}

export type CustomImportValidationResponse = {
    year: number;
    count: number;
    sample: TxnSample;
    valid: boolean;
    categories: string[];
}

export type Import = {
    user_id: number;
    account_id: number;
    name: string;
    status: string;
    import_type: string;
    currency: string;
    started_at: Date;
    completed_at: Date | null;
}