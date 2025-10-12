import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";

export const useDataStore = defineStore('data', {
    state: () => ({
        importPrefix: "imports",
    }),
    getters: {
    },
    actions: {
        async validateImport(importType: string, record: object) {
            try {
                const response = await apiClient.post(`${this.importPrefix}/${importType}/validate`, record);
                return response.data;
            } catch (err) {
                throw err;
            }
        },

        async importFromJSON(file: File) {
            const form = new FormData();
            form.append("file", file);
            const { data } = await apiClient.post(
                `${this.importPrefix}/custom/json`,
                form,
                { headers: { 'Content-Type': 'multipart/form-data' } }
            );
            return data;
        },

    },
});
