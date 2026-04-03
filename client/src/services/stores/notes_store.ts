import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";

export const useNotesStore = defineStore("notes", {
  state: () => ({
    apiPrefix: "notes",
  }),
  getters: {},
  actions: {
    async toggleResolveState(id: number) {
      return await apiClient.post(`${this.apiPrefix}/${id}/resolve`);
    },
  },
});
