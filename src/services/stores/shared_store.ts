import { defineStore } from 'pinia';
import apiClient from "../api/axios_interceptor.ts";

export const useSharedStore = defineStore('shared', {
    actions: {
        async getRecordsPaginated(prefix: string, params: object, page: number) {
            try {
                const queryParams = {
                    ...params,
                    page: page,
                };

                const response = await apiClient.get(`${prefix}`, {
                    params: queryParams,
                });

                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async getRecordByID(prefix: string, id: number, params?: object) {
            try {

                const response = await apiClient.get(`${prefix}/${id}`, {
                    params: params
                });
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async createRecord(prefix: string, record: object) {
            try {
                const response = await apiClient.put(`${prefix}`, record);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async updateRecord(prefix: string, id: number, record: object) {
            try {
                const response = await apiClient.put(`${prefix}/${id}`, record);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
        async deleteRecord(prefix: string, id: number) {
            try {
                const response = await apiClient.delete(`${prefix}/${id}`);
                return response.data;
            } catch (err) {
                throw err;
            }
        },
    },
});
