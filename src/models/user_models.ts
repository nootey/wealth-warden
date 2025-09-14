export interface User {
    id?: number;
    display_name: string;
    email: string;
    email_confirmed?: Date | null;
    role_id?: number;
    role?: Role;
    deleted_at?: Date | null;
}

export interface Role {
    id?: number;
    name: string;
    description?: string | null;
    permissions?: Permission[];
}

export interface Permission {
    id: number;
    name: string;
    description: string | null;
}