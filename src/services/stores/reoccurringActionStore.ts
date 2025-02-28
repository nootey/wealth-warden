import {defineStore} from "pinia";
import type {ReoccurringAction} from "../../models/actions.ts";
import apiClient from "./api/axios_interceptor.ts";

export const useActionStore = defineStore('action', {
    state: () => ({
        reoccurringActions: [] as ReoccurringAction[],
    }),
    actions: {
        async getAllActionsForCategory(categoryName: string) {
            try {
                const response = await apiClient.get("get-all-reoccurring-actions-for-category", {params: {categoryName: categoryName}});
                this.reoccurringActions = response.data;
            } catch (err) {
                throw err;
            }
        },

        async deleteRecAction(id: number, categoryName: string) {
            try {
                const response = await apiClient.post("delete-reoccurring-action", {id: id, category_name: categoryName});
                await this.getAllActionsForCategory(categoryName);
                return response;
            } catch (err) {
                throw err;
            }
        },
    }
});