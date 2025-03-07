import {defineStore} from "pinia";
import type {ReoccurringAction} from "../../models/actions.ts";
import apiClient from "./api/axios_interceptor.ts";

export const useActionStore = defineStore('action', {
    state: () => ({
        apiPrefix: "reoccurring",
        reoccurringActions: [] as ReoccurringAction[],
    }),
    actions: {
        async getAllActionsForCategory(categoryName: string) {
            try {
                const response = await apiClient.get(`${this.apiPrefix}/by-category`, {params: {categoryName: categoryName}});
                this.reoccurringActions = response.data;
            } catch (err) {
                throw err;
            }
        },

        async deleteRecAction(id: number, categoryName: string) {
            try {
                const response = await apiClient.post(`${this.apiPrefix}/delete`, {id: id, category_name: categoryName});
                await this.getAllActionsForCategory(categoryName);
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});