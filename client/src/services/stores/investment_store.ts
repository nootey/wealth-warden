import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useInvestmentStore = defineStore("investment", {
  state: () => ({
    apiPrefix: "investments",
  }),
  getters: {},
  actions: {
    async getAllAssets() {
      const response = await apiClient.get(`${this.apiPrefix}/all`);
      return response.data;
    },
    async createIncome(req: object) {
      const response = await apiClient.put(`${this.apiPrefix}/income`, req);
      return response.data;
    },
    async deleteIncome(id: number) {
      const response = await apiClient.delete(`${this.apiPrefix}/income/${id}`);
      return response.data;
    },
  },
});
