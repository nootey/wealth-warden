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

        async getCustomImportJSON(id: number | string, step: string): Promise<any> {
            try {
                const response = await apiClient.get(`${this.importPrefix}/custom/${id}?step=${encodeURIComponent(step)}`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },

        async validateImport(importType: string, record: object, importStep: string) {
            try {
                const response = await apiClient.post(
                    `${this.importPrefix}/${importType}/validate?step=${encodeURIComponent(importStep)}`,
                    record
                );
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
        },

        async transferInvestmentsFromImport(payload: {
            import_id: number | string
            investment_mappings: { name: string; account_id: number | null }[]
        }) {
            return await apiClient.post(`${this.importPrefix}/custom/json/investments`, payload)
        },

    },
});
