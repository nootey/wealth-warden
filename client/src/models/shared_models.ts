export interface FilterObj {
    source: string;
    field: string | null;
    operator: string;
    value: unknown;
    display?: string;
}

export interface SortObj {
    order: number;
    field: string;
}

export type Operator = { name: string; value: string };