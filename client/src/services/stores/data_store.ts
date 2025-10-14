import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";

export const useDataStore = defineStore('data', {
    state: () => ({
        importPrefix: "imports",
    }),
    getters: {
    },
    actions: {
        async getImports(importType: string) {
            try {
                const res = await apiClient.get(`${this.importPrefix}/${importType}`);
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async validateImport(importType: string, record: object) {
            try {
                const response = await apiClient.post(`${this.importPrefix}/${importType}/validate`, record);
                return response.data;
            } catch (err) {
                throw err;
            }
        },

        async importFromJSON(payload: any, checkID: number) {
            const { data } = await apiClient.post(
                `${this.importPrefix}/custom/json`,
                payload,
                { params: { check_acc_id: checkID } }
            );
            return data;
        }

    },
});
