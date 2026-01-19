import { defineStore } from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    apiPrefix: "settings",
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
    async updatePreferenceSettings(settings: object) {
      return await apiClient.put(
        `${this.apiPrefix}/users/preferences`,
        settings,
      );
    },
    async updateProfileSettings(settings: object) {
      return await apiClient.put(`${this.apiPrefix}/users/profile`, settings);
    },

    async createDatabaseDump() {
      const res = await apiClient.post(`${this.apiPrefix}/backups/create`);
      return res.data;
    },
  },
});
