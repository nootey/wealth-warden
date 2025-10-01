import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";

export const useChartStore = defineStore('chart', {
    state: () => ({
        apiPrefix: "charts",
    }),
    getters: {
    },
    actions: {
        async getNetWorth(params?: {
            range?: string; from?: string; to?: string; currency?: string;
            account?: number | string | null;
        }) {
            try {
                const q: Record<string, any> = {}
                if (params) {
                    for (const [k, v] of Object.entries(params)) {
                        if (v !== undefined && v !== null && v !== '') q[k] = v
                    }
                }

                const response = await apiClient.get(`${this.apiPrefix}/networth`, { params: q })
                return response.data
            } catch (err) {
                throw err
            }
        },
        async getMonthlyCashFlowForYear(params?: {
            year: number;
            account?: number | string | null;
        }) {
            try {
                const q: Record<string, any> = {}
                if (params) {
                    for (const [k, v] of Object.entries(params)) {
                        if (v !== undefined && v !== null && v !== '') q[k] = v
                    }
                }

                const response = await apiClient.get(`${this.apiPrefix}/monthly-cash-flow`, { params: q })
                return response.data
            } catch (err) {
                throw err
            }
        }
    },
});
