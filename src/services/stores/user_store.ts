import {defineStore} from "pinia";
import type {AuthForm} from "../../models/auth_models.ts";
import apiClient from "../api/axios_interceptor.ts";
import type {Role} from "../../models/user_models.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
        roles: [] as Role[],
    }),
    actions: {
        async getRoles(withPerms: boolean = false) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/roles`, {
                    params: {withPerms}
                });
                this.roles = response.data;
            } catch (err) {
                throw err;
            }
        },
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
