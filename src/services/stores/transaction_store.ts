import { defineStore } from 'pinia';
import apiClient from "../api/axios_interceptor.ts";

export const useTransactionStore = defineStore('transaction', {
    state: () => ({
        apiPrefix: "transactions",
    }),
    getters: {
    },
    actions: {
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
        async getFilterData() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/filter-data`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async createTransaction(transactionData: object) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}`, transactionData);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async updateTransaction(id: number, transactionData: object) {
            try {
                const response = await apiClient.put(`${this.apiPrefix}/${id}`, transactionData);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async deleteTransaction(id: number) {
            try {
                const response = await apiClient.delete(`${this.apiPrefix}/${id}`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
    },
});
