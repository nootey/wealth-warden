import {defineStore} from "pinia";
import apiClient from '../api/axios_interceptor.ts';
import type {SavingsCategory} from "../../models/savings.ts";

export const useSavingsStore = defineStore('savings', {
    state: () => ({
        apiPrefix: "savings",
        savingsCategories: [] as SavingsCategory[],
        currentYear: new Date().getFullYear(),
        savingsYears: [] as number[],
    }),
    actions: {

        async getSavingsYears() {
            try {
                const response = await apiClient.get("reoccurring/available-record-years", {
                    params: {table: "savings_allocations", field: "savings_date"}});
                this.savingsYears = response.data;
            } catch (err) {
                throw err;
            }
        },

        async getSavingsPaginated(params: object, page: number) {
            try {

                const queryParams = {
                    ...params,
                    page: page,
                };

                const response = await apiClient.get(`${this.apiPrefix}/`, {
                    params: queryParams,
                });

                return response.data;

            } catch (err) {
                throw err;
            }
        },

        async getSavingsCategories() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories`);
                this.savingsCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async createSavingsCategory(SavingsCategory: SavingsCategory|null) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/create-category`, SavingsCategory);
                await this.getSavingsCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

    }
});