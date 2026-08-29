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
    apiPrefix: "backoffice/jobs",
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

      const response = await apiClient.get(this.apiPrefix, {
        params: queryParams,
      });
      return response.data;
    },
    async getCounts(): Promise<JobCounts> {
      const response = await apiClient.get(`${this.apiPrefix}/counts`);
      return response.data.data;
    },
    async getJob(id: number): Promise<RiverJobDetail> {
      const response = await apiClient.get(`${this.apiPrefix}/${id}`);
      return response.data.data;
    },
    async getQueues(): Promise<RiverQueue[]> {
      const response = await apiClient.get(`${this.apiPrefix}/queues`);
      return response.data.data ?? [];
    },
    async getPeriodic(): Promise<RiverPeriodicJob[]> {
      const response = await apiClient.get(`${this.apiPrefix}/periodic`);
      return response.data.data ?? [];
    },
    async retryJobs(ids: number[]) {
      const response = await apiClient.post(`${this.apiPrefix}/retry`, { ids });
      return response.data;
    },
    async cancelJobs(ids: number[]) {
      const response = await apiClient.post(`${this.apiPrefix}/cancel`, {
        ids,
      });
      return response.data;
    },
    async deleteJobs(ids: number[]) {
      const response = await apiClient.post(`${this.apiPrefix}/delete`, {
        ids,
      });
      return response.data;
    },
    async pauseQueue(name: string) {
      const response = await apiClient.post(
        `${this.apiPrefix}/queues/${encodeURIComponent(name)}/pause`,
      );
      return response.data;
    },
    async resumeQueue(name: string) {
      const response = await apiClient.post(
        `${this.apiPrefix}/queues/${encodeURIComponent(name)}/resume`,
      );
      return response.data;
    },
  },
});
