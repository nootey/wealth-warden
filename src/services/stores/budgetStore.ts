import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {MonthlyBudget} from "../../models/budgets.ts";

export const useBudgetStore = defineStore('budget', {
    state: () => ({
        apiPrefix: "budget",
        current_budget: null
    }),
    actions: {
        async getCurrentBudget() {
            try {
                return await apiClient.get(`${this.apiPrefix}/current`);
            } catch (error) {
                throw error;
            }
        },

        async createNewBudget(Budget: MonthlyBudget|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create`, Budget);
            } catch (err) {
                throw err;
            }
        },

        async updateBudgetSnapshot(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/update-snapshot`, {id: id});
            } catch (err) {
                throw err;
            }
        },
    }
});