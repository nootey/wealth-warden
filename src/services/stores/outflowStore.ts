import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';
import type {Outflow, OutflowCategory} from "../../models/outflows.ts";
import type {ReoccurringAction} from "../../models/actions.ts";

export const useOutflowStore = defineStore('outflow', {
    state: () => ({
        outflowCategories: [] as OutflowCategory[],
        currentYear: new Date().getFullYear(),
        outflowYears: [] as number[],
    }),
    actions: {

        async getOutflowYears() {
            try {
                const response = await apiClient.get("get-available-record-years", {
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

                const response = await apiClient.get("get-outflows-paginated", {
                    params: queryParams,
                });

                return response.data;

            } catch (err) {
                throw err;
            }
        },

        async getAllGroupedOutflows(year: number) {
            try {

                return await apiClient.get("get-all-outflows-grouped-month", {params: {year: year}});

            } catch (err) {
                throw err;
            }
        },

        async getOutflowCategories() {
            try {
                const response = await apiClient.get("get-all-outflow-categories");
                this.outflowCategories = response.data;
            } catch (err) {
                throw err;
            }
        },

        async createOutflow(Outflow: Outflow|null) {
            try {
                return await apiClient.post("create-new-outflow", Outflow);
            } catch (err) {
                throw err;
            }
        },

        async updateOutflow(Outflow: Outflow|null) {
            try {
                return await apiClient.post("update-outflow", Outflow);
            } catch (err) {
                throw err;
            }
        },

        async createReoccurringOutflow(Outflow: Outflow|null, RecOutflow: ReoccurringAction|null) {
            try {
                return await apiClient.post("create-new-reoccurring-outflow", {Outflow, RecOutflow});
            } catch (err) {
                throw err;
            }
        },

        async createOutflowCategory(OutflowCategory: OutflowCategory|null) {
            try {
                const response = await apiClient.post("create-new-outflow-category", OutflowCategory);
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async updateOutflowCategory(OutflowCategory: OutflowCategory|null) {
            try {
                const response = await apiClient.post("update-outflow-category", OutflowCategory);
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },

        async deleteOutflow(id: number) {
            try {
                return await apiClient.post("delete-outflow", {id: id});
            } catch (err) {
                throw err;
            }
        },

        async deleteOutflowCategory(id: number) {
            try {
                const response = await apiClient.post("delete-outflow-category", {id: id});
                await this.getOutflowCategories();
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});