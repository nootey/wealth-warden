import {defineStore} from "pinia";
import apiClient from '../api/axios_interceptor.ts';
import type {SavingsTransaction, SavingsCategory} from "../../models/savings.ts";
import type {ReoccurringAction} from "../../models/actions.ts";

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
                    params: {table: "savings_transactions", field: "transaction_date"}});
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

                const response = await apiClient.get(`${this.apiPrefix}`, {
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

        async getAllGroupedSavings(year: number) {
            try {

                return await apiClient.get(`${this.apiPrefix}/grouped-by-month`, {params: {year: year}});

            } catch (err) {
                throw err;
            }
        },

        async createSavingsAllocation(Allocation: SavingsTransaction|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-allocation`, Allocation);
            } catch (err) {
                throw err;
            }
        },

        async createSavingsDeduction(Deduction: SavingsTransaction|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-deduction`, Deduction);
            } catch (err) {
                throw err;
            }
        },

        async createSavingsCategory(SavingsCategory: SavingsCategory|null, IsReoccurring: boolean, RecAction: ReoccurringAction|null, Allocation: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/create-category`, {
                    category: SavingsCategory,
                    is_reoccurring: IsReoccurring,
                    reoccurring_action: RecAction,
                    allocated_amount: Allocation
                });
                await this.getSavingsCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async updateSavingsCategory(SavingsCategory: SavingsCategory|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/update-category`, SavingsCategory);
            } catch (err) {
                throw err;
            }
        },
        async deleteSavingsCategory(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/delete-category`, {id: id});
            } catch (err) {
                throw err;
            }
        },

    }
});