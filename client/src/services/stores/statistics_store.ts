import { defineStore } from "pinia";
import apiClient from "../api/axios.ts";
import type {
  BasicAccountStats,
  DailyStats,
  MonthlyStats,
  YearlyBreakdownStats,
} from "../../models/statistics_models.ts";

export const useStatisticsStore = defineStore("statistics", {
  state: () => ({
    apiPrefix: "statistics",
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
        `/statistics/categories/${id}/average`,
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
  },
});
