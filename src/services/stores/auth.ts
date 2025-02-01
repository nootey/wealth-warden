import { defineStore } from 'pinia';
import apiClient from '../../api';

interface AuthForm {
    email: string;
    password: string;
}

export const useAuthStore = defineStore('auth', {
    state: () => ({
        authenticated: false,
    }),
    actions: {
        async login(authForm: AuthForm) {
            try {
                const response = await apiClient.post('/login', authForm);
                // Example: assume the API returns a token
                this.authenticated = true;
                return response;
            } catch (error) {
                console.error('Login failed:', error);
                throw error;
            }
        },
    },
});
