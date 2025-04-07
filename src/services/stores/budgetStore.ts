import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {MonthlyBudget, MonthlyBudgetAllocation} from "../../models/budgets.ts";

export const useBudgetStore = defineStore('budget', {
    state: () => ({
        apiPrefix: "budget",
        current_budget: null as any
    }),
    getters: {
        getAllocationByIndex: (state) => (index: string) => {
            if (!state.current_budget || !state.current_budget.allocations) return null;
            return state.current_budget.allocations.find(
                (allocation: any) => allocation.category === index
            );
        }
    },
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
            if (this.current_budget !== null) return this.current_budget

            try {
                const response = await apiClient.get(`${this.apiPrefix}/current`)
                this.current_budget = response.data
                return this.current_budget
            } catch (error) {
                console.error('Failed to fetch current budget:', error)
                throw error
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