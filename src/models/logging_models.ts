export interface Causer {
    id: number;
    username: string;
}

export interface ActivityLog {
    event: string | null;
    category: string | null;
    causer: object | null;
    metadata: object | null;
}