import { defineStore } from "pinia";
import apiClient from "../api/axios.ts";

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
    async syncAssetPrice(id: number) {
      const response = await apiClient.get(`${this.apiPrefix}/sync/${id}`);
      return response.data;
    },
    async syncAssetAccountBalance(acc_id: number) {
      const response = await apiClient.get(
        `${this.apiPrefix}/sync/account/${acc_id}`,
      );
      return response.data;
    },
  },
});
