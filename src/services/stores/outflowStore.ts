import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';
import type {Outflow, OutflowCategory} from "../../models/outflows.ts";
import type {ReoccurringAction} from "../../models/actions.ts";

export const useOutflowStore = defineStore('outflow', {
    state: () => ({
        apiPrefix: "outflows",
        outflowCategories: [] as OutflowCategory[],
        currentYear: new Date().getFullYear(),
        outflowYears: [] as number[],
    }),
    actions: {

        async getOutflowYears() {
            try {
                const response = await apiClient.get("reoccurring/available-record-years", {
                    params: {table: "outflows", field: "outflow_date"}});
                this.outflowYears = response.data;
            } catch (err) {
                throw err;
            }
        },

        async getOutflowsPaginated(params: object, page: number) {
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

        async getAllGroupedOutflows(year: number) {
            try {

                return await apiClient.get(`${this.apiPrefix}/grouped-by-month`, {params: {year: year}});

            } catch (err) {
                throw err;
            }
        },

        async getOutflowCategories() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/categories`);
                this.outflowCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async createOutflow(Outflow: Outflow|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create`, Outflow);
            } catch (err) {
                throw err;
            }
        },

        async updateOutflow(Outflow: Outflow|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/update`, Outflow);
            } catch (err) {
                throw err;
            }
        },

        async createReoccurringOutflow(Outflow: Outflow|null, RecOutflow: ReoccurringAction|null) {
            try {
                return await apiClient.post(`${this.apiPrefix}/create-reoccurring`, {Outflow, RecOutflow});
            } catch (err) {
                throw err;
            }
        },

        async createOutflowCategory(OutflowCategory: OutflowCategory|null) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/create-category`, OutflowCategory);
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async updateOutflowCategory(OutflowCategory: OutflowCategory|null) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/update-category`, OutflowCategory);
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async deleteOutflow(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/delete`, {id: id});
            } catch (err) {
                throw err;
            }
        },

        async deleteOutflowCategory(id: number) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/delete-category`, {id: id});
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});