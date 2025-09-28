import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";
import type {BasicAccountStats} from "../../models/statistics_models.ts";

export const useStatisticsStore = defineStore('statistics', {
    state: () => ({
        apiPrefix: 'statistics',
    }),
    actions: {
        async getBasicStatisticsForAccount(accID: number | null | undefined, year: number) {
            const res = await apiClient.get<BasicAccountStats>("statistics/account", {
                params: {
                    year,
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        },
        async getAvailableStatsYears(accID: number | null | undefined) {
            const res = await apiClient.get<number[]>("statistics/years", {
                params: {
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        }
    },
});
