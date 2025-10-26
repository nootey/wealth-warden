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
    filtered_count: number;
    sample: TxnSample;
    step: string;
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

export type CustomExportValidationResponse = {
    txn_count: number;
    transfer_count: number;
    acc_count: number;
    category_count: number;
}

export type Export = {
    user_id: number;
    account_id: number;
    name: string;
    status: string;
    export_type: string;
    currency: string;
    started_at: Date;
    completed_at: Date | null;
}