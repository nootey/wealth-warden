import {defineStore} from "pinia";

export const useUserStore = defineStore('user', {
    state: () => ({
        apiPrefix: "users",
    }),
    actions: {

    },
});
