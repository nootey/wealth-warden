import {defineStore} from "pinia";
import type {AuthForm} from "../../models/auth_models.ts";
import apiClient from "../api/axios_interceptor.ts";
import type {Permission, Role} from "../../models/user_models.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
        roles: [] as Role[],
        permissions: [] as Permission[],
    }),
    actions: {
        async getRoles(with_permissions: boolean = false) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/roles`, {
                    params: {with_permissions}
                });
                this.roles = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getPermissions() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/permissions`);
                this.permissions = response.data;
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
        async getInvitationByHash(hash: string) {
            try {
                const res = await apiClient.get(`${this.apiPrefix}/invitations/${hash}`);
                return res.data;
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
        async resendInvitation(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/invitations/resend/${id}`);
            } catch (error) {
                throw error;
            }
        },
    },
});
