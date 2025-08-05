export interface FilterObj {
    parameter: string | null;
    operator: string;
    value: any;
}

export interface SortObj {
    order: number;
    field: string;
}

export type Operator = { name: string; value: string };