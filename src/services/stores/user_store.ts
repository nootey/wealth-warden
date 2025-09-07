import {defineStore} from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
    }),
    actions: {

    },
});
