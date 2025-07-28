export interface User {
    id: number;
    username: string;
    display_name: string;
    email: string;
    validated_at?: string;
    role_id: number;
}