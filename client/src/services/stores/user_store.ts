import { defineStore } from "pinia";
import type { AuthForm } from "../../models/auth_models.ts";
import apiClient from "../api/axios_interceptor.ts";
import type { Permission, Role } from "../../models/user_models.ts";

export const useUserStore = defineStore("user", {
  state: () => ({
    apiPrefix: "users",
    roles: [] as Role[],
    permissions: [] as Permission[],
  }),
  actions: {
    async getRoles(with_permissions: boolean = false) {
      const response = await apiClient.get(`${this.apiPrefix}/roles`, {
        params: { with_permissions },
      });
      this.roles = response.data;
    },
    async getPermissions() {
      const response = await apiClient.get(`${this.apiPrefix}/permissions`);
      this.permissions = response.data;
    },
    async getUserByToken(tokenType: string, tokenValue: string) {
      return await apiClient.get(`${this.apiPrefix}/token`, {
        params: {
          type: tokenType,
          value: tokenValue,
        },
      });
    },
    async getInvitationByHash(hash: string) {
      const res = await apiClient.get(`${this.apiPrefix}/invitations/${hash}`);
      return res.data;
    },
    async createInvitation(authForm: AuthForm) {
      return await apiClient.put(`${this.apiPrefix}/invitations`, authForm);
    },
    async resendInvitation(id: number) {
      return await apiClient.post(`${this.apiPrefix}/invitations/resend/${id}`);
    },
  },
});
