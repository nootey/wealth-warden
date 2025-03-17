import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {MonthlyBudget, MonthlyBudgetAllocation} from "../../models/budgets.ts";

export const useBudgetStore = defineStore('budget', {
    state: () => ({
        apiPrefix: "budget",
        current_budget: null
    }),
    actions: {
        async synchronizeMonthlyBudget() {
            try {
                return await apiClient.get(`${this.apiPrefix}/sync`);
            } catch (error) {
                throw error;
            }
        },

        async synchronizeMonthlyBudgetSnapshot() {
            try {
                return await apiClient.get(`${this.apiPrefix}/sync-snapshot`);
            } catch (error) {
                throw error;
            }
        },

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

        async createNewBudgetAllocation(Allocation: MonthlyBudgetAllocation|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-allocation`, Allocation);
            } catch (err) {
                throw err;
            }
        },

        async updateMonthlyBudget(budgetID: number, field: string, value: any) {
            try {
                return await apiClient.post(`${this.apiPrefix}/update`, {budget_id: budgetID, field: field, value: value});
            } catch (err) {
                throw err;
            }
        },

    }
});