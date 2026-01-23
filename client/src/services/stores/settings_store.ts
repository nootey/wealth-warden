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

    async getBackups() {
      return await apiClient.get(`${this.apiPrefix}/backups`);
    },

    async createDatabaseDump() {
      const res = await apiClient.post(`${this.apiPrefix}/backups/create`);
      return res.data;
    },

    async restoreFromDatabaseDump(backup_name: string) {
      const res = await apiClient.post(`${this.apiPrefix}/backups/restore`, {
        backup_name: backup_name,
      });
      return res.data;
    },

    async downloadBackup(backup_name: string) {
      const res = await apiClient.post(
        `${this.apiPrefix}/backups/download`,
        { backup_name: backup_name },
        {
          responseType: "blob",
        },
      );

      const blob = new Blob([res.data], { type: "application/zip" });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${backup_name}.zip`;
      document.body.appendChild(a);
      a.click();
      a.remove();
    },
  },
});
