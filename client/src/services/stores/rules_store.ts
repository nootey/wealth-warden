import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";
import type { Rule } from "../../models/rules_models.ts";

export const useRulesStore = defineStore("rules", {
  state: () => ({
    apiPrefix: "rules",
    rules: [] as Rule[],
  }),
  getters: {},
  actions: {
    async getRules() {
      const response = await apiClient.get(`${this.apiPrefix}`);
      this.rules = response.data;
    },
  },
});
