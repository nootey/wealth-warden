import {defineStore} from 'pinia';
import apiClient from "./api/axios_interceptor.ts";

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
    }
});