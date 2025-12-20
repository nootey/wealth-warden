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
  },
});
