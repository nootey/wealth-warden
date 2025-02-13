import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';

interface InflowCategory {
    name: string;
}

interface Inflow {
    inflow_category_id: number;
    inflow_category: object;
    amount: number;
    inflow_date: any;
}

export const useGeneralStore = defineStore('general', {
    state: () => ({}),
    actions: {
        async getInflowsPaginated() {
            try {
                const response = await apiClient.get("get-inflows-paginated");
                return response.data;
            } catch (err) {
                console.error(err);
                throw err;
            }
        },

        async getInflowCategories() {
            try {
                const response = await apiClient.get("get-all-inflow-categories");
                return response.data;
            } catch (err) {
                console.error(err);
                throw err;
            }
        },

        async createInflow(Inflow: Inflow|null) {
            try {
                return await apiClient.post("create-new-inflow", Inflow);
            } catch (err) {
                console.error(err);
                throw err;
            }
        },

        async createInflowCategory(InflowCategory: InflowCategory|null) {
            try {
                return await apiClient.post("create-new-inflow-category", InflowCategory);
            } catch (err) {
                console.error(err);
                throw err;
            }
        }
    }
});