export interface User {
    id: number;
    username: string;
    display_name: string;
    email: string;
    email_confirmed?: Date | null;
    role_id: number;
}