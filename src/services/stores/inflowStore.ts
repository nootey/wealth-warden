import {defineStore} from "pinia";
import apiClient from '../api/axios_interceptor.ts';
import type {Inflow, InflowCategory} from "../../models/inflows.ts";
import type {ReoccurringAction} from "../../models/actions.ts";
import type {DynamicCategory, DynamicCategoryMapping} from "../../models/shared.ts";

export const useInflowStore = defineStore('inflow', {
    state: () => ({
        apiPrefix: "inflows",
        inflowCategories: [] as InflowCategory[],
        dynamicCategories: [] as DynamicCategory[],
        currentYear: new Date().getFullYear(),
        inflowYears: [] as number[],
    }),
    actions: {

        async getInflowYears() {
            try {
                const response = await apiClient.get("reoccurring/available-record-years", {
                    params: {table: "inflows", field: "inflow_date"}});
                this.inflowYears = response.data;
            } catch (err) {
                throw err;
            }
        },

        async getInflowsPaginated(params: object, page: number) {
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

        async getAllGroupedInflows(year: number) {
            try {

                return await apiClient.get(`${this.apiPrefix}/grouped-by-month`, {params: {year: year}});

            } catch (err) {
                throw err;
            }
        },

        async getInflowCategories() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories`);
                this.inflowCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async getDynamicCategories() {
            try {
                const response =  await apiClient.get(`${this.apiPrefix}/dynamic-categories`);
                this.dynamicCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async createInflow(Inflow: Inflow|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create`, Inflow);
            } catch (err) {
                throw err;
            }
        },

        async updateInflow(Inflow: Inflow|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/update`, Inflow);
            } catch (err) {
                throw err;
            }
        },

        async createReoccurringInflow(Inflow: Inflow|null, RecInflow: ReoccurringAction|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-reoccurring`, {Inflow, RecInflow});
            } catch (err) {
                throw err;
            }
        },

        async createInflowCategory(InflowCategory: InflowCategory|null) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/create-category`, InflowCategory);
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async createDynamicCategory(Category: DynamicCategory, Mapping: DynamicCategoryMapping) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-dynamic-category`, {Category, Mapping});
            } catch (err) {
                throw err;
            }
        },

        async updateInflowCategory(InflowCategory: InflowCategory|null) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/update-category`, InflowCategory);
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async deleteInflow(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/delete`, {id: id});
            } catch (err) {
                throw err;
            }
        },

        async deleteInflowCategory(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/delete-category`, {id: id});
                await this.getInflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async deleteDynamicCategory(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/delete-dynamic-category`, {id: id});
                await this.getDynamicCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});