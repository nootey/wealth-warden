import {defineStore} from 'pinia';
import type {Account, AccountType} from '../../models/account_models.ts'
import apiClient from "../api/axios.ts";

export const useAccountStore = defineStore('account', {
    state: () => ({
        apiPrefix: "accounts",
        accountTypes: [] as AccountType[],
        accounts: [] as Account[],
    }),
    getters: {
    },
    actions: {
        async getAllAccounts(toReturn: boolean = false, includeTypes: boolean = false) {
            const params = includeTypes ? '?types=true' : '';
            const response = await apiClient.get(`${this.apiPrefix}/all${params}`);
            if(toReturn){
                return response.data;
            } else {
                this.accounts = response.data;
            }
        },
        async getAccountTypes() {
            const response = await apiClient.get(`${this.apiPrefix}/types`);
            this.accountTypes = response.data;
        },
        async getAccountsBySubtype(subtype: string) {
            const response = await apiClient.get(`${this.apiPrefix}/subtype/${subtype}`);
            return response.data;
        },
        async getAccountsByType(type: string) {
            const response = await apiClient.get(`${this.apiPrefix}/type/${type}`);
            return response.data;
        },
        async toggleActiveState(id: number) {
            return await apiClient.post(`${this.apiPrefix}/${id}/active`);
        },
        async backfillBalances() {
            return await apiClient.post(`${this.apiPrefix}/balances/backfill`);
        },
        async getNetWorth(currency = "EUR") {
            const response = await apiClient.get(`${this.apiPrefix}/charts/networth`, {
                params: { range: "1m", currency }
            });
            return response.data;
        },
        async saveProjection(id: number, record: object) {
            return await apiClient.post(`${this.apiPrefix}/${id}/projection/save`, record);
        },
        async revertProjection(id: number) {
            return await apiClient.post(`${this.apiPrefix}/${id}/projection/revert`);
        },
        async getLatestBalance(id: number) {
            const response = await apiClient.get(`${this.apiPrefix}/balances/${id}/latest`);
            return response.data;
        },
        async getAllDefaultAccounts() {
            const response = await apiClient.get(`${this.apiPrefix}/defaults/all`);
            return response.data;
        },
        async getAccountTypesWithoutDefaults() {
            const response = await apiClient.get(`${this.apiPrefix}/defaults/types`);
            return response.data;
        },
        async setDefaultAccount(accountId: number) {
            const response = await apiClient.patch(`${this.apiPrefix}/defaults/set/${accountId}`);
            return response.data;
        },
        async unsetDefaultAccount(accountId: number) {
            const response = await apiClient.patch(`${this.apiPrefix}/defaults/unset/${accountId}`);
            return response.data;
        }
    },
});
