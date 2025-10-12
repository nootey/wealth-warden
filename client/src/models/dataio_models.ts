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
}