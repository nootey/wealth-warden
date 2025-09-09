import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";

export const useChartStore = defineStore('chart', {
    state: () => ({
        apiPrefix: "charts",
    }),
    getters: {
    },
    actions: {
        async getNetWorth(currency = "EUR") {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/networth`, {
                    params: { range: "1m", currency }
                });
                return response.data;
            } catch (err) {
                throw err;
            }
        }
    },
});
