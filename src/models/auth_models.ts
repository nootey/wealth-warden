export interface AuthForm {
    email: string;
    password: string;
    passwordConfirmation?: string;
    rememberMe?: boolean;
}