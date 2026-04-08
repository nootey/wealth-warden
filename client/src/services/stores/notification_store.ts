import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useNotificationStore = defineStore("notifications", {
  state: () => ({
    apiPrefix: "notifications",
    hasUnread: false,
  }),
  actions: {
    async markAsRead(id: number) {
      return await apiClient.post(`notifications/${id}/read`);
    },
    async markAllAsRead() {
      return await apiClient.post("notifications/read-all");
    },
    async checkUnread() {
      try {
        const response = await apiClient.get("notifications", {
          params: { rowsPerPage: 1, unread: true, page: 1 },
        });
        this.hasUnread = (response.data?.total_records ?? 0) > 0;
      } catch {
        // non-critical
      }
    },
  },
});
