export interface AuthForm {
    email: string;
    password: string;
    rememberMe?: boolean;
}

export interface User {
    id: number;
    email: string;
    validated_at?: string;
}