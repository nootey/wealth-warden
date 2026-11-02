import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useLoggingStore = defineStore("logging", {
  state: () => ({
    apiPrefix: "logs",
  }),
  actions: {
    async getLogsPaginated(params: object, cursor: string | null) {
      const queryParams = {
        ...params,
        cursor: cursor ?? undefined,
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
      cursor: string | null,
      rowsPerPage: number,
    ) {
      if (!events || events.length === 0) {
        return {
          data: [],
          rows_per_page: rowsPerPage,
          total_records: 0,
          next_cursor: null,
          prev_cursor: null,
        };
      }

      const response = await apiClient.get(`${this.apiPrefix}/audit-trail`, {
        params: {
          id: id.toString(),
          event: events.join(","),
          category: categories.join(","),
          cursor: cursor ?? undefined,
          rowsPerPage: rowsPerPage,
        },
      });

      return response.data;
    },
  },
});
