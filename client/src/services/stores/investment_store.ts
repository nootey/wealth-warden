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
  },
});
