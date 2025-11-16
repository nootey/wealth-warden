import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";
import type {Category, CategoryGroup} from "../../models/transaction_models.ts";

export const useTransactionStore = defineStore('transaction', {
    state: () => ({
        apiPrefix: "transactions",
        currentYear: new Date().getFullYear(),
        categories: [] as Category[],
        category_groups: [] as CategoryGroup[],
    }),
    getters: {
    },
    actions: {
        async getPaginatedTransactionsForAccount(params: object, page: number, accID: number) {
            try {

                const queryParams = {
                    ...params,
                    page: page,
                    account: accID,
                };

                const response = await apiClient.get(`${this.apiPrefix}`, {
                    params: queryParams,
                });

                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async getCategories(deleted: boolean = false) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories`, {
                    params: {deleted}
                });
                this.categories = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getCategoryGroups() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories/groups`);
                this.category_groups = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getCategoriesWithGroups() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories/groups/all`);
                return response.data;
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
        async restoreTransaction(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/restore`, { id });
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async restoreCategory(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/categories/restore`, { id });
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async restoreCategoryName(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/categories/restore/name`, { id });
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async toggleTemplateActiveState(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/templates/${id}/active`);
            } catch (err) {
                throw err;
            }
        },
        async getTransactionTemplateCount() {
            try {
                return await apiClient.get(`${this.apiPrefix}/templates/count`);
            } catch (err) {
                throw err;
            }
        },
    },
});
