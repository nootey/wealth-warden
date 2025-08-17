import { defineStore } from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {Category} from "../../models/transaction_models.ts";
import type {Account} from "../../models/account_models.ts";

export const useTransactionStore = defineStore('transaction', {
    state: () => ({
        apiPrefix: "transactions",
        currentYear: new Date().getFullYear(),
        categories: [] as Category[],
        accounts: [] as Account[],
    }),
    getters: {
    },
    actions: {
        async getCategories() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories`);
                this.categories = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getAccounts() {
            try {
                const response = await apiClient.get(`accounts/all`);
                this.accounts = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getTransactionsPaginated(params: object, page: number) {
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
    },
});
