export interface AccountType {
    id: number|null;
    name: string;
    type: string;
    subtype: string;
    classification: string;
}

export interface Balance{
    id: number|null;
    as_of: Date | null;
    start_balance: number | null;
    end_balance: number | null;
}

export interface Account {
    id: number|null;
    name: string;
    account_type: AccountType,
    balance: Balance;
}