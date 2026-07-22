import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

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
      categories: string[],
      page: number,
      rowsPerPage: number,
    ) {
      if (!events || events.length === 0) {
        return { data: [], total_records: 0, from: 0, to: 0 };
      }

      const response = await apiClient.get(`${this.apiPrefix}/audit-trail`, {
        params: {
          id: id.toString(),
          event: events.join(","),
          category: categories.join(","),
          page: page,
          rowsPerPage: rowsPerPage,
        },
      });

      return response.data;
    },
  },
});
