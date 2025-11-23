import {defineStore} from 'pinia';
import apiClient from "../api/axios_interceptor.ts";

export const useDataStore = defineStore('data', {
    state: () => ({
        importPrefix: "imports",
        exportPrefix: "exports",
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

        async importTransactions(payload: any, checkID: number) {
            const { data } = await apiClient.post(
                `${this.importPrefix}/custom/transactions`,
                payload,
                { params: { check_acc_id: checkID } }
            );
            return data;
        },

        async importAccounts(payload: any, useBalances: boolean) {
            const { data } = await apiClient.post(
                `${this.importPrefix}/custom/accounts`,
                payload,
                { params: { use_balances: useBalances } }
            );
            return data;
        },

        async importCategories(payload: any) {
            const { data } = await apiClient.post(
                `${this.importPrefix}/custom/categories`,
                payload
            );
            return data;
        },

        async transferInvestmentsFromImport(payload: {
            import_id: number | string
            checking_acc_id: number
            investment_mappings: { name: string; account_id: number | null }[]
        }) {
            try {
                const res = await apiClient.post(
                    `${this.importPrefix}/custom/investments`,
                    payload
                );
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async transferSavingsFromImport(payload: {
            import_id: number | string
            checking_acc_id: number
            savings_mappings: { name: string; account_id: number | null }[]
        }) {
            try {
                const res = await apiClient.post(
                    `${this.importPrefix}/custom/savings`,
                    payload
                );
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async transferRepaymentsFromImport(payload: {
            import_id: number | string
            checking_acc_id: number
            repayment_mappings: { name: string; account_id: number | null }[]
        }) {
            try {
                const res = await apiClient.post(
                    `${this.importPrefix}/custom/repayments`,
                    payload
                );
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async getExports() {
            try {
                const res = await apiClient.get(`${this.exportPrefix}`);
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async exportData() {
            try {
                const res = await apiClient.post(`${this.exportPrefix}`);
                return res.data;
            } catch (err) {
                throw err;
            }
        },

        async downloadExport(id: number) {
            try {
                const res = await apiClient.post(`${this.exportPrefix}/${id}/download`, null, {
                    responseType: 'blob',
                });

                const blob = new Blob([res.data], { type: 'application/zip' });
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'export.zip';
                document.body.appendChild(a);
                a.click();
                a.remove();
            } catch (err) {
                console.error('Export failed', err);
                throw err;
            }
        },

    },
});
