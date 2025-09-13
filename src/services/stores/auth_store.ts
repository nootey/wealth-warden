import {defineStore} from 'pinia';
import apiClient from '../api/axios_interceptor.ts';
import router from "../router/main.ts";
import type {AuthForm} from '../../models/auth_models.ts';
import type {User} from '../../models/user_models.ts';
import {watch} from "vue";

export const useAuthStore = defineStore('auth', {
    state: () => ({
        apiPrefix: "auth",
        authenticated: localStorage.getItem('authenticated') == "true",
        user: null as User | null,
        initialized: false,
    }),
    getters: {
        isAuthenticated: (s) => s.authenticated,
        isInitialized:  (s) => s.initialized,
        isValidated:     (s) => !!s.user?.email_confirmed,
    },
    actions: {
        async login(form: AuthForm) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/login`, form);
                await this.init();
                return response;
            } catch (error) {
                throw error;
            }
        },

        async signUp(form: AuthForm) {
            try {
                return await apiClient.post(`${this.apiPrefix}/signup`, form);
            } catch (error) {
                throw error;
            }
        },

        async resendConfirmationEmail(email?: string) {
            try {
                return await apiClient.post(`${this.apiPrefix}/resend-confirmation-email`, {email: email});
            } catch (error) {
                console.error(error)
            }
        },

        async getAuthUser(set = true) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/current`, {params: {withSecrets: true}});

                if (set) {
                    if (!response.data) {
                        await this.logoutUser();
                    } else {
                        this.setUser(response.data);
                    }
                }

                return response.data;
            } catch (error) {
                await this.logoutUser();
                throw error;
            }
        },

        setUser(userData: User) {
            this.user = userData;
        },

        async logoutUser() {
            try {
                await apiClient.post(`${this.apiPrefix}/logout`, null);
            } catch (error) {
            }
            this.logout();
        },

        logout() {
            this.user = null;
            this.setAuthenticated(false);
            this.setInitialized(null);
            
            const darkModeActive = localStorage.getItem('darkModeActive');
            localStorage.clear();
            if (darkModeActive !== null) {
                localStorage.setItem('darkModeActive', darkModeActive);
            }
            
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
        },

        async waitForUser(): Promise<User> {
            // If the user is already loaded, return immediately.
            if (this.user !== null) return this.user;

            // Otherwise, watch for changes to the user property.
            return new Promise((resolve) => {
                const stopWatch = watch(
                    () => this.user,
                    (newUser) => {
                        if (newUser !== null) {
                            stopWatch(); // Stop watching once user is set.
                            resolve(newUser);
                        }
                    }
                );
            });
        },

    },
});
