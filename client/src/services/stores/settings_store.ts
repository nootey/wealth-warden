import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    apiPrefix: "settings",
    defaultCurrency: "EUR",
  }),
  actions: {
    async getGeneralSettings() {
      return await apiClient.get(`${this.apiPrefix}`);
    },
    async getUserSettings() {
      return await apiClient.get(`${this.apiPrefix}/users`);
    },
    async getAvailableTimezones() {
      return await apiClient.get(`${this.apiPrefix}/timezones`);
    },
    async getAvailableCurrencies() {
      return await apiClient.get(`${this.apiPrefix}/currencies`);
    },
    async updatePreferenceSettings(settings: object) {
      return await apiClient.put(
        `${this.apiPrefix}/users/preferences`,
        settings,
      );
    },
    async updateProfileSettings(settings: object) {
      return await apiClient.put(`${this.apiPrefix}/users/profile`, settings);
    },
  },
});
