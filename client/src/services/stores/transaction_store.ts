import { defineStore } from "pinia";
import apiClient from "../api/axios_interceptor.ts";
import type {
  Category,
  CategoryGroup,
} from "../../models/transaction_models.ts";

export const useTransactionStore = defineStore("transaction", {
  state: () => ({
    apiPrefix: "transactions",
    currentYear: new Date().getFullYear(),
    categories: [] as Category[],
    category_groups: [] as CategoryGroup[],
  }),
  getters: {},
  actions: {
    async getPaginatedTransactionsForAccount(
      params: object,
      page: number,
      accID: number,
    ) {
      const queryParams = {
        ...params,
        page: page,
        account: accID,
      };

      const response = await apiClient.get(`${this.apiPrefix}`, {
        params: queryParams,
      });

      return response.data;
    },
    async getCategories(deleted: boolean = false) {
      const response = await apiClient.get(`${this.apiPrefix}/categories`, {
        params: { deleted },
      });
      this.categories = response.data;
    },
    async getCategoryGroups() {
      const response = await apiClient.get(
        `${this.apiPrefix}/categories/groups`,
      );
      this.category_groups = response.data;
    },
    async getCategoriesWithGroups() {
      const response = await apiClient.get(
        `${this.apiPrefix}/categories/groups/all`,
      );
      return response.data;
    },
    async startTransfer(record: object) {
      const response = await apiClient.put(
        `${this.apiPrefix}/transfers`,
        record,
      );
      return response.data;
    },
    async restoreTransaction(id: number) {
      const response = await apiClient.post(`${this.apiPrefix}/restore`, {
        id,
      });
      return response.data;
    },
    async restoreCategory(id: number) {
      const response = await apiClient.post(
        `${this.apiPrefix}/categories/restore`,
        { id },
      );
      return response.data;
    },
    async restoreCategoryName(id: number) {
      const response = await apiClient.post(
        `${this.apiPrefix}/categories/restore/name`,
        { id },
      );
      return response.data;
    },
    async toggleTemplateActiveState(id: number) {
      return await apiClient.post(`${this.apiPrefix}/templates/${id}/active`);
    },
    async getTransactionTemplateCount() {
      return await apiClient.get(`${this.apiPrefix}/templates/count`);
    },
  },
});
