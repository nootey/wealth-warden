import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";
import type {
  SavingGoalWithProgress,
  SavingContribution,
  SavingGoalReq,
  SavingGoalUpdateReq,
  SavingContributionReq,
} from "../../models/savings_models.ts";

export const useSavingsStore = defineStore("savings", {
  state: () => ({
    apiPrefix: "savings",
  }),
  getters: {},
  actions: {
    async fetchGoals(): Promise<SavingGoalWithProgress[]> {
      const response = await apiClient.get<SavingGoalWithProgress[]>(
        `${this.apiPrefix}`,
      );
      return response.data;
    },

    async fetchGoalByID(id: number): Promise<SavingGoalWithProgress> {
      const response = await apiClient.get<SavingGoalWithProgress>(
        `${this.apiPrefix}/${id}`,
      );
      return response.data;
    },

    async insertGoal(req: SavingGoalReq) {
      return await apiClient.put(`${this.apiPrefix}`, req);
    },

    async updateGoal(id: number, req: SavingGoalUpdateReq) {
      return await apiClient.put(`${this.apiPrefix}/${id}`, req);
    },

    async deleteGoal(id: number) {
      return await apiClient.delete(`${this.apiPrefix}/${id}`);
    },

    async fetchContributions(goalID: number): Promise<SavingContribution[]> {
      const response = await apiClient.get<SavingContribution[]>(
        `${this.apiPrefix}/${goalID}/contributions`,
      );
      return response.data;
    },

    async insertContribution(goalID: number, req: SavingContributionReq) {
      return await apiClient.put(
        `${this.apiPrefix}/${goalID}/contributions`,
        req,
      );
    },

    async deleteContribution(goalID: number, contribID: number) {
      return await apiClient.delete(
        `${this.apiPrefix}/${goalID}/contributions/${contribID}`,
      );
    },
  },
});
