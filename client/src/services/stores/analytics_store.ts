import { defineStore } from "pinia";
import apiClient from "../api/axios.ts";
import type {
    BasicAccountStats,
    DailyStats,
    MonthlyStats,
    YearlyBreakdownStats,
} from "../../models/analytics_models.ts";

export const useAnalyticsStore = defineStore("analytics", {
    state: () => ({
        apiPrefix: "analytics",
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
        async getBasicStatisticsForAccount(
            accID: number | null | undefined,
            year: number,
        ) {
            const res = await apiClient.get<BasicAccountStats>(
                `${this.apiPrefix}/account`,
                {
                    params: {
                        year,
                        acc_id: accID ?? undefined,
                    },
                },
            );
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
        async getTodayStats(accID: number | null | undefined) {
            const res = await apiClient.get<DailyStats>(`${this.apiPrefix}/today`, {
                params: {
                    acc_id: accID ?? undefined,
                },
            });
            return res.data;
        },
        async getCategoryAverage(
            id: number,
            account_id: number,
            isGroup: boolean,
        ): Promise<number> {
            const response = await apiClient.get(
                `${this.apiPrefix}/categories/${id}/average`,
                {
                    params: {
                        account_id: account_id,
                        is_group: isGroup,
                    },
                },
            );
            return response.data.average;
        },

        async getYearlyBreakdownStats(
            accID: number | null,
            year: number,
            comparisonYear: number | null = null,
        ) {
            const params: any = { year };
            if (accID) params.acc_id = accID;
            if (comparisonYear) params.comparison_year = comparisonYear;

            return await apiClient.get<YearlyBreakdownStats>(
                `${this.apiPrefix}/breakdown/yearly`,
                { params },
            );
        },
        async getNetWorth(params?: {
            range?: string;
            from?: string;
            to?: string;
            currency?: string;
            account?: number | string | null;
        }) {
            const q: Record<string, unknown> = {};
            if (params) {
                for (const [k, v] of Object.entries(params)) {
                    if (v !== undefined && v !== null && v !== "") q[k] = v;
                }
            }

            const response = await apiClient.get(`${this.apiPrefix}/networth`, {
                params: q,
            });
            return response.data;
        },

        async getMonthlyCashFlowForYear(params?: {
            year: number;
            account?: number | string | null;
        }) {
            const q: Record<string, unknown> = {};
            if (params) {
                for (const [k, v] of Object.entries(params)) {
                    if (v !== undefined && v !== null && v !== "") q[k] = v;
                }
            }

            const response = await apiClient.get(
                `${this.apiPrefix}/monthly-cash-flow`,
                { params: q },
            );
            return response.data;
        },

        async getYearlyCashFlowOverviewForYear(params?: {
            year: number;
            account?: number | string | null;
        }) {
            const q: Record<string, unknown> = {};
            if (params) {
                for (const [k, v] of Object.entries(params)) {
                    if (v !== undefined && v !== null && v !== "") q[k] = v;
                }
            }

            const response = await apiClient.get(
                `${this.apiPrefix}/yearly-cash-flow-breakdown`,
                { params: q },
            );
            return response.data;
        },

        async getMultiYearMonthlyCategoryBreakdown(params: {
            years: number[];
            account?: number | string | null;
            class?: "income" | "expense";
            percent?: boolean;
            category?: number | string | null;
        }) {
            const q: Record<string, unknown> = {};
            q["years"] = params.years.join(",");
            if (params.account != null && params.account !== "")
                q["account"] = params.account;
            if (params.class) q["class"] = params.class;
            if (typeof params.percent === "boolean") q["percent"] = params.percent;
            if (params.category != null && params.category !== "")
                q["category"] = params.category;

            const response = await apiClient.get(
                `${this.apiPrefix}/monthly-category-breakdown`,
                { params: q },
            );
            return response.data;
        },

        async getYearlySankeyData(params: {
            year: number;
            account?: number | null;
        }) {
            const response = await apiClient.get(`${this.apiPrefix}/sankey`, {
                params,
            });
            return response.data;
        },
    },
});
