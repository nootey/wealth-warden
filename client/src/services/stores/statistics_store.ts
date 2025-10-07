import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";
import type {BasicAccountStats, MonthlyStats} from "../../models/statistics_models.ts";

export const useStatisticsStore = defineStore('statistics', {
    state: () => ({
        apiPrefix: 'statistics',
    }),
    actions: {
        async getAvailableStatsYears(accID: number | null | undefined) {
            const res = await apiClient.get<number[]>(`${this.apiPrefix}/years`, {
                params: {
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        },
        async getBasicStatisticsForAccount(accID: number | null | undefined, year: number) {
            const res = await apiClient.get<BasicAccountStats>(`${this.apiPrefix}/account`, {
                params: {
                    year,
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        },
        async getCurrentMonthsStats(accID: number | null | undefined) {
            const res = await apiClient.get<MonthlyStats>(`${this.apiPrefix}/month`, {
                params: {
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        },
    },
});
