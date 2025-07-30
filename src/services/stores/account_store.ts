import { defineStore } from 'pinia';
import type {AccountType} from '../../models/account_models.ts'
import apiClient from "../api/axios.ts";

export const useAccountStore = defineStore('account', {
    state: () => ({
        apiPrefix: "accounts",
        accountTypes: [] as AccountType[],
    }),
    getters: {
    },
    actions: {
        async getAccountTypes() {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/types`);
                this.accountTypes = response.data;
            } catch (err) {
                throw err;
            }
        },
    },
});
