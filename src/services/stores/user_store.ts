import {defineStore} from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
    }),
    actions: {
        async getUserSettings() {
            try {
                return await apiClient.get(`settings/users`);
            } catch (err) {
                throw err;
            }
        },
    },
});
