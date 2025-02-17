import { defineStore } from 'pinia';
import apiClient from './api/axios_interceptor.ts';
import router from "../router";
import type {AuthForm, User} from '../../models/auth.ts';

export const useAuthStore = defineStore('auth', {
    state: () => ({
        authenticated: localStorage.getItem('authenticated') == "true",
        user: null as User | null,
        initialized: false,
    }),
    getters: {
        isAuthenticated: (state) => state.authenticated,
        isInitialized: (state) => state.initialized,
    },
    actions: {
        async login(authForm: AuthForm) {
            try {
                const response = await apiClient.post('/login', authForm);
                await this.init();
                return response;
            } catch (error) {
                console.error('Login failed:', error);
                throw error;
            }
        },

        async getAuthUser(set = true) {
            try {
                const response = await apiClient.get('/get-auth-user');

                if (set) {
                    if (!response.data) {
                        await this.logoutUser();
                    } else {
                        this.setUser(response.data);
                    }
                }

                return response.data;
            } catch (error) {
                console.error('Auth user not found:', error);
                await this.logoutUser();
                throw error;
            }
        },

        setUser(userData: User) {
            this.user = userData;
        },

        async logoutUser() {
            try {
                await apiClient.post('/logout', null);
            } catch (error) {
                console.error('Logout failed:', error);
            }
            this.logout();
        },

        logout() {
            this.user = null;
            this.setAuthenticated(false);
            this.setInitialized(null);
            localStorage.clear();
            sessionStorage.clear();
            router.push("/login").then()
        },

        setAuthenticated(status: boolean) {
            this.authenticated = status;
            localStorage.setItem("authenticated", status.toString());
        },

        setInitialized(user: User | null) {
            this.initialized = user !== null;
        },

        async init() {
            try {
                this.setAuthenticated(true);
                const user = await this.getAuthUser(true);
                if (user) {
                    this.setInitialized(user);
                } else {
                    this.setAuthenticated(false);
                    this.setInitialized(null);
                }
            } catch (error) {
                this.setAuthenticated(false);
                this.setInitialized(null);
            }
        }
    },
});
