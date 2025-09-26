import {defineStore} from 'pinia';
import apiClient from "../api/axios.ts";
import type {BasicAccountStats} from "../../models/statistics_models.ts";

export const useStatisticsStore = defineStore('statistics', {
    state: () => ({
        apiPrefix: 'statistics',
    }),
    actions: {
        async getBasicStatisticsForAccount(accID: number, year: number) {
            const res = await apiClient.get<BasicAccountStats>(
                `${this.apiPrefix}/account/${accID}`,
                { params: { year } }
            );
            return res.data;
        },
    },
});
