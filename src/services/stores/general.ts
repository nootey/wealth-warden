import {defineStore} from "pinia";
import apiClient from './api/axios_interceptor.ts';

interface InflowType {
    name: string;
}

export const useGeneralStore = defineStore('general', {
    state: () => ({}),
    actions: {
        async getInflowTypes() {
            try {
                const response = await apiClient.get("get-all-inflow-types");
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