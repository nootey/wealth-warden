import { defineStore } from "pinia";
import apiClient from "../api/api_client.ts";
import type {
  JobCounts,
  JobListParams,
  RiverJobDetail,
  RiverPeriodicJob,
  RiverQueue,
} from "../../models/job_models.ts";

export const useJobsStore = defineStore("jobs", {
  state: () => ({
    adminApiPrefix: "jobs/admin",
    userApiPrefix: "jobs/user",
  }),
  actions: {
    async getJobsPaginated(params: JobListParams, page: number) {
      const queryParams: Record<string, unknown> = {
        page,
        rowsPerPage: params.rowsPerPage,
        sort: params.sort,
      };
      if (params.states.length > 0) queryParams.state = params.states.join(",");
      if (params.filters.length > 0) queryParams.filters = params.filters;

      const response = await apiClient.get(this.adminApiPrefix, {
        params: queryParams,
      });
      return response.data;
    },
    async getCounts(): Promise<JobCounts> {
      const response = await apiClient.get(`${this.adminApiPrefix}/counts`);
      return response.data.data;
    },
    async getJob(id: number): Promise<RiverJobDetail> {
      const response = await apiClient.get(`${this.adminApiPrefix}/${id}`);
      return response.data.data;
    },
    async getQueues(): Promise<RiverQueue[]> {
      const response = await apiClient.get(`${this.adminApiPrefix}/queues`);
      return response.data.data ?? [];
    },
    async getPeriodic(): Promise<RiverPeriodicJob[]> {
      const response = await apiClient.get(`${this.adminApiPrefix}/periodic`);
      return response.data.data ?? [];
    },
    async retryJobs(ids: number[]) {
      const response = await apiClient.post(`${this.adminApiPrefix}/retry`, {
        ids,
      });
      return response.data;
    },
    async cancelJobs(ids: number[]) {
      const response = await apiClient.post(`${this.adminApiPrefix}/cancel`, {
        ids,
      });
      return response.data;
    },
    async deleteJobs(ids: number[]) {
      const response = await apiClient.post(`${this.adminApiPrefix}/delete`, {
        ids,
      });
      return response.data;
    },
    async pauseQueue(name: string) {
      const response = await apiClient.post(
        `${this.adminApiPrefix}/queues/${encodeURIComponent(name)}/pause`,
      );
      return response.data;
    },
    async resumeQueue(name: string) {
      const response = await apiClient.post(
        `${this.adminApiPrefix}/queues/${encodeURIComponent(name)}/resume`,
      );
      return response.data;
    },
    async listJobs(kind: string, rowsPerPage: number, page: number) {
      const response = await apiClient.get(this.userApiPrefix, {
        params: { kind, rowsPerPage, page },
      });
      return response.data;
    },
    async startJob(kind: string) {
      const response = await apiClient.post(`${this.userApiPrefix}/start`, {
        kind,
      });
      return response.data;
    },
    async retryJob(id: number) {
      const response = await apiClient.post(
        `${this.userApiPrefix}/${id}/retry`,
      );
      return response.data;
    },
    async cancelJob(id: number) {
      const response = await apiClient.post(
        `${this.userApiPrefix}/${id}/cancel`,
      );
      return response.data;
    },
  },
});
