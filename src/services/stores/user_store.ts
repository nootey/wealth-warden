import {defineStore} from "pinia";
import type {AuthForm} from "../../models/auth_models.ts";
import apiClient from "../api/axios_interceptor.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
    }),
    actions: {
        async createInvitation(authForm: AuthForm) {
            try {
                return await apiClient.put(`${this.apiPrefix}/invitations`, authForm);
            } catch (error) {
                throw error;
            }
        },
    },
});
