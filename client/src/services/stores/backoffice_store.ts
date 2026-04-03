import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useBackofficeStore = defineStore("backoffice", {
  state: () => ({
    apiPrefix: "backoffice",
  }),
  getters: {},
  actions: {
    async backFillAssetCashflow() {
      const response = await apiClient.post(
        `${this.apiPrefix}/backfill/asset-cash-flows`,
      );
      return response.data;
    },
  },
});
