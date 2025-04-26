import {defineStore} from "pinia";
import apiClient from "../api/axios_interceptor.ts";

export const useLoggingStore = defineStore('logging', {
    state: () => ({
        apiPrefix: "logs",
    }),
    actions: {
        async getLogsPaginated(logType: string, params: object, page: number) {
            try {

                const queryParams = {
                    ...params,
                    page: page,
                };

                console.log(queryParams);

                const response = await apiClient.get(`${this.apiPrefix}/${logType}`, {
                    params: queryParams,
                });

                return response.data;

            } catch (err) {
                throw err;
            }
        },
        async getFilterData(index: string) {
            try {

                const response = await apiClient.get(`${this.apiPrefix}/filter-data`, {
                    params: {index: index},
                });

                return response.data;

            } catch (err) {
                throw err;
            }
        },
    }
});