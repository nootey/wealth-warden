import { defineStore } from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {Category} from "../../models/transaction_models.ts";

export const useTransactionStore = defineStore('transaction', {
    state: () => ({
        apiPrefix: "transactions",
        currentYear: new Date().getFullYear(),
        categories: [] as Category[],
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
        async startTransfer(record: object) {
            try {
                const response = await apiClient.put(`${this.apiPrefix}/transfers`, record);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
    },
});
