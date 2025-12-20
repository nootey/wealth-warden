import { defineStore } from "pinia";
import apiClient from "../api/axios.ts";

export const useChartStore = defineStore("chart", {
  state: () => ({
    apiPrefix: "charts",
  }),
  getters: {},
  actions: {
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
  },
});
