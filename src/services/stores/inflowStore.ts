import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';
import type {Inflow, InflowCategory} from "../../models/inflows.ts";
import type {ReoccurringAction} from "../../models/actions.ts";
import type {DynamicCategory, DynamicCategoryMapping} from "../../models/shared.ts";

export const useInflowStore = defineStore('inflow', {
    state: () => ({
        inflowCategories: [] as InflowCategory[],
    }),
    actions: {
        async getInflowsPaginated(params: object, page: number) {
            try {

                const queryParams = {
                    ...params,
                    page: page,
                };

                const response = await apiClient.get("get-inflows-paginated", {
                    params: queryParams,
                });

                return response.data;

            } catch (err) {
                throw err;
            }
        },

        async getAllGroupedInflows() {
            try {

                return await apiClient.get("get-all-inflows-grouped-month");

            } catch (err) {
                throw err;
            }
        },

        async getInflowCategories() {
            try {
                const response = await apiClient.get("get-all-inflow-categories");
                this.inflowCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async getDynamicCategories() {
            try {
                return await apiClient.get("get-all-dynamic-categories");
            } catch (err) {
                throw err;
            }
        },

        async createInflow(Inflow: Inflow|null) {
            try {
                return await apiClient.post("create-new-inflow", Inflow);
            } catch (err) {
                throw err;
            }
        },

        async updateInflow(Inflow: Inflow|null) {
            try {
                return await apiClient.post("update-inflow", Inflow);
            } catch (err) {
                throw err;
            }
        },

        async createReoccurringInflow(Inflow: Inflow|null, RecInflow: ReoccurringAction|null) {
            try {
                return await apiClient.post("create-new-reoccurring-inflow", {Inflow, RecInflow});
            } catch (err) {
                throw err;
            }
        },

        async createInflowCategory(InflowCategory: InflowCategory|null) {
            try {
                const response = await apiClient.post("create-new-inflow-category", InflowCategory);
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async createDynamicCategory(Category: DynamicCategory, Mapping: DynamicCategoryMapping) {
            try {
                return await apiClient.post("create-new-dynamic-category", {Category, Mapping});
            } catch (err) {
                throw err;
            }
        },

        async updateInflowCategory(InflowCategory: InflowCategory|null) {
            try {
                const response = await apiClient.post("update-inflow-category", InflowCategory);
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async deleteInflow(id: number) {
            try {
                return await apiClient.post("delete-inflow", {id: id});
            } catch (err) {
                throw err;
            }
        },

        async deleteInflowCategory(id: number) {
            try {
                const response = await apiClient.post("delete-inflow-category", {id: id});
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});