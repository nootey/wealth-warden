import { defineStore } from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useSharedStore = defineStore("shared", {
  actions: {
    async getRecordsPaginated(prefix: string, params: object, page: number) {
      const queryParams = {
        ...params,
        page: page,
      };

      const response = await apiClient.get(`${prefix}`, {
        params: queryParams,
      });

      return response.data;
    },
    async getRecordByID(prefix: string, id: number, params?: object) {
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
    async deleteRecord(prefix: string, id: number, params?: object) {
      const response = await apiClient.delete(`${prefix}/${id}`, {
        params: params,
      });
      return response.data;
    },
  },
});
