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
    async getTaxBrackets() {
      const response = await apiClient.get(`${this.apiPrefix}/tax-brackets`);
      return response.data;
    },
    async createTaxBracket(req: object) {
      const response = await apiClient.put(
        `${this.apiPrefix}/tax-brackets`,
        req,
      );
      return response.data;
    },
    async deleteTaxBracket(id: number) {
      const response = await apiClient.delete(
        `${this.apiPrefix}/tax-brackets/${id}`,
      );
      return response.data;
    },
    async getTaxSettings() {
      const response = await apiClient.get(`${this.apiPrefix}/tax-settings`);
      return response.data;
    },
    async saveTaxSettings(req: object) {
      const response = await apiClient.put(
        `${this.apiPrefix}/tax-settings`,
        req,
      );
      return response.data;
    },
    async copyTaxBrackets(fromType: string, toType: string) {
      const response = await apiClient.post(
        `${this.apiPrefix}/tax-brackets/copy`,
        {
          from_type: fromType,
          to_type: toType,
        },
      );
      return response.data;
    },
  },
});
