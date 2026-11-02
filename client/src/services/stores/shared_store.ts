import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useSharedStore = defineStore("shared", {
  actions: {
    async getRecordsPaginated(
      prefix: string,
      params: Record<string, unknown>,
      cursor: string | null = null,
    ) {
      const queryParams = {
        ...params,
        cursor: cursor ?? undefined,
      };

      const response = await apiClient.get(`${prefix}`, {
        params: queryParams,
      });

      return response.data;
    },
    async getRecordByID(
      prefix: string,
      id: number,
      params?: Record<string, unknown>,
    ) {
      const response = await apiClient.get(`${prefix}/${id}`, {
        params: params,
      });
      return response.data;
    },
    async createRecord(prefix: string, record: object) {
      const response = await apiClient.put(`${prefix}`, record);
      return response.data;
    },
    async updateRecord(prefix: string, id: number, record: object) {
      const response = await apiClient.put(`${prefix}/${id}`, record);
      return response.data;
    },
    async deleteRecord(
      prefix: string,
      id: number,
      params?: Record<string, unknown>,
    ) {
      const response = await apiClient.delete(`${prefix}/${id}`, {
        params: params,
      });
      return response.data;
    },
  },
});
