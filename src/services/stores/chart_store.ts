import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";

export const useChartStore = defineStore('chart', {
    state: () => ({
        apiPrefix: "charts",
    }),
    getters: {
    },
    actions: {
        async getNetWorth(params?: { range?: string; from?: string; to?: string; currency?: string }) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/networth`, {
                    params: { params }
                });
                return response.data;
            } catch (err) {
                throw err;
            }
        }
    },
});
