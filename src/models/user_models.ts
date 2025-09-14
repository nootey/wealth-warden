export interface User {
    id: number;
    username: string;
    display_name: string;
    email: string;
    email_confirmed?: Date | null;
    role_id: number;
    role?: Role;
}

export interface Role {
    id: number;
    name: string;
    description: string | null;
    permissions: RolePermissions[];
}

export interface Permission {
    id: number;
    name: string;
    description: string | null;
}

export interface RolePermissions {
    id: number;
    role_id: number;
    permission_id: number;
    permission: Permission;
}