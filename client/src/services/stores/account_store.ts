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
        async getAllAccounts(toReturn: boolean = false) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/all`);
                if(toReturn){
                    return response.data;
                } else {
                    this.accounts = response.data;
                }
            } catch (err) {
                throw err;
            }
        },
        async getAccountTypes() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/types`);
                this.accountTypes = response.data;
            } catch (err) {
                throw err;
            }
        },
        async getAccountsBySubtype(subtype: string) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/subtype/${subtype}`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async getAccountsByType(type: string) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/type/${type}`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async toggleActiveState(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/${id}/active`);
            } catch (err) {
                throw err;
            }
        },
        async backfillBalances() {
            try {
                return await apiClient.post(`${this.apiPrefix}/balances/backfill`);
            } catch (err) {
                throw err;
            }
        },
        async getNetWorth(currency = "EUR") {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/charts/networth`, {
                    params: { range: "1m", currency }
                });
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async saveProjection(id: number, record: object) {
            try {
                return await apiClient.post(`${this.apiPrefix}/${id}/projection/save`, record);
            } catch (err) {
                throw err;
            }
        },
        async revertProjection(id: number) {
            try {
                return await apiClient.post(`${this.apiPrefix}/${id}/projection/revert`);
            } catch (err) {
                throw err;
            }
        },
    },
});
