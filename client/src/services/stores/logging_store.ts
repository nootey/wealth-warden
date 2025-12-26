import { defineStore } from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useLoggingStore = defineStore("logging", {
  state: () => ({
    apiPrefix: "logs",
  }),
  actions: {
    async getLogsPaginated(params: object, page: number) {
      const queryParams = {
        ...params,
        page: page,
      };

      const response = await apiClient.get(`${this.apiPrefix}`, {
        params: queryParams,
      });

      return response.data;
    },
    async getFilterData(index: string) {
      const response = await apiClient.get(`${this.apiPrefix}/filter-data`, {
        params: { index: index },
      });

      return response.data;
    },
    async getAuditTrail(
      id: string | number,
      events: string[],
      category: string,
    ) {
      if (!events || events.length === 0) {
        return { data: [] }; // Return empty result gracefully
      }

      const response = await apiClient.get(`${this.apiPrefix}/audit-trail`, {
        params: {
          id: id.toString(),
          event: events.join(","),
          category: category,
        },
      });

      return response.data;
    },
  },
});
