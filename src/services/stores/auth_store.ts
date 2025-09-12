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
        isAuthenticated: (state) => state.authenticated,
        isInitialized: (state) => state.initialized,
    },
    actions: {
        async register(authForm: AuthForm) {
            try {
                return await apiClient.post(`${this.apiPrefix}/register`, authForm);
            } catch (error) {
                throw error;
            }
        },

        async login(authForm: AuthForm) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/login`, authForm);
                await this.init();
                return response;
            } catch (error) {
                throw error;
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
