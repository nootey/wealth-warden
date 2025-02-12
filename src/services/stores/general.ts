import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';

interface InflowType {
    name: string;
}

interface Inflow {
    inflow_type_id: number;
    inflow_type: object;
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

        async getInflowTypes() {
            try {
                const response = await apiClient.get("get-all-inflow-types");
                return response.data;
            } catch (err) {
                console.error(err);
                throw err;
            }
        },

        async createInflow(Inflow: Inflow|null) {
            try {
                console.log(Inflow)
                const response = await apiClient.post("create-new-inflow", Inflow);
                return response.data;
            } catch (err) {
                console.error(err);
                throw err;
            }
        },

        async createInflowType(InflowType: InflowType|null) {
            try {
                const response = await apiClient.post("create-new-inflow-type", InflowType);
                return response.data;
            } catch (err) {
                console.error(err);
                throw err;
            }
        }
    }
});