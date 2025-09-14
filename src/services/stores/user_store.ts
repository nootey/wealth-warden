import {defineStore} from "pinia";
import type {AuthForm} from "../../models/auth_models.ts";
import apiClient from "../api/axios_interceptor.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
    }),
    actions: {
        async getUserByToken(tokenType: string, tokenValue: string) {
            try {
                return await apiClient.get(`${this.apiPrefix}/token`, {
                    params: {
                        type: tokenType,
                        value: tokenValue,
                    },
                });
            } catch (error) {
                throw error;
            }
        },
        async createInvitation(authForm: AuthForm) {
            try {
                return await apiClient.put(`${this.apiPrefix}/invitations`, authForm);
            } catch (error) {
                throw error;
            }
        },
    },
});
