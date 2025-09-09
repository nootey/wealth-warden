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
        async getAllAccounts() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/all`);
                this.accounts = response.data;
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
    },
});
