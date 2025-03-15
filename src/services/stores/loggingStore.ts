import {defineStore} from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useLoggingStore = defineStore('logging', {
    state: () => ({
        apiPrefix: "logs",
    }),
    actions: {
        async getActivityLogs() {
            try {
                const response = await apiClient.get('/get-activity-logs');
                return response.data;
            } catch (err) {
                throw err;
            }
        },
    }
});