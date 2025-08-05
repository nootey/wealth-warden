export interface Causer {
    id: number;
    username: string;
}

export interface ActivityLogFilterData {
    events: string[];
    categories: string[];
    causers: Causer[];
}

export type FilterValue = string | Causer;