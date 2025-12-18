import {defineStore} from 'pinia';
import apiClient from '../api/axios_interceptor.ts';
import router from "../router/main.ts";
import type {AuthForm} from '../../models/auth_models.ts';
import type {User} from '../../models/user_models.ts';
import {watch} from "vue";
import {useSettingsStore} from "./settings_store.ts";
import {useThemeStore} from "./theme_store.ts";

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
        isAdmin: (s) => s.user?.role?.name == "super-admin" || s.user?.role?.name == "admin",
        isSuperAdmin: (s) => s.user?.role?.name == "super-admin",
    },
    actions: {
        async login(form: AuthForm) {
            const response = await apiClient.post(`${this.apiPrefix}/login`, form);
            await this.init();
            return response;
        },

        async signUp(form: AuthForm, invitation_id: number | null = null) {
            return await apiClient.post(`${this.apiPrefix}/signup`, {
                ...form,
                ...(invitation_id && { invitation_id })
            });
        },

        async resendConfirmationEmail(email?: string) {
            return await apiClient.post(`${this.apiPrefix}/resend-confirmation-email`, {email: email});
        },

        async requestPasswordReset(email?: string) {
            return await apiClient.post(`${this.apiPrefix}/request-password-reset`, {email: email});
        },

        async resetPassword(form: AuthForm) {
            return await apiClient.post(`${this.apiPrefix}/reset-password`, form);
        },

        async getAuthUser(set = true) {
            const response = await apiClient.get(`${this.apiPrefix}/current`, {params: {withSecrets: true}});

            if (set) {
                if (!response.data) {
                    await this.logoutUser();
                } else {
                    this.setUser(response.data);
                }
            }

            return response.data;
        },

        setUser(userData: User) {
            this.user = userData;
        },

        async logoutUser() {
            await apiClient.post(`${this.apiPrefix}/logout`, null);
            this.logout();
        },

        logout() {
            this.user = null;
            this.setAuthenticated(false);
            this.setInitialized(null);

            localStorage.clear();
            sessionStorage.clear();

            const themeStore = useThemeStore();
            themeStore.setTheme('dark');

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

                    const settingsStore = useSettingsStore();
                    const themeStore = useThemeStore();

                    try {
                        const response = await settingsStore.getUserSettings();
                        if (response.data) {
                            themeStore.setTheme(
                                response.data.theme || 'system',
                                response.data.accent || 'blurple'
                            );
                        }
                    } catch (error) {
                        console.error('Failed to load theme settings:', error);
                    }


                } else {
                    this.setAuthenticated(false);
                    this.setInitialized(null);
                }
            } catch {
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
