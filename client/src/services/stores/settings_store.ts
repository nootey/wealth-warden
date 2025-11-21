import {defineStore} from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useSettingsStore = defineStore('settings', {
    state: () => ({
        apiPrefix: "settings",
    }),
    actions: {
        async getGeneralSettings() {
            try {
                return await apiClient.get(`${this.apiPrefix}`);
            } catch (err) {
                throw err;
            }
        },
        async getUserSettings() {
            try {
                return await apiClient.get(`${this.apiPrefix}/users`);
            } catch (err) {
                throw err;
            }
        },
        async getAvailableTimezones() {
            try {
                return await apiClient.get(`${this.apiPrefix}/timezones`);
            } catch (err) {
                throw err;
            }
        },
        async updatePreferenceSettings(settings: object) {
            try {
                return await apiClient.put(`${this.apiPrefix}/users/preferences`, settings);
            } catch (err) {
                throw err;
            }
        },
        async updateProfileSettings(settings: object) {
            try {
                return await apiClient.put(`${this.apiPrefix}/users/profile`, settings);
            } catch (err) {
                throw err;
            }
        },
    },
});
