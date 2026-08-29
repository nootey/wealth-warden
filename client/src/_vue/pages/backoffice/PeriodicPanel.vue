<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useJobsStore } from "../../../services/stores/jobs_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import dateHelper from "../../../utils/date_helper.ts";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import type { Column } from "../../../services/filter_registry.ts";
import type { RiverPeriodicJob } from "../../../models/job_models.ts";

const jobsStore = useJobsStore();
const toastStore = useToastStore();

const loading = ref(false);
const jobs = ref<RiverPeriodicJob[]>([]);

const activeColumns = computed<Column[]>(() => [
  { field: "kind", header: "Kind" },
  { field: "schedule", header: "Schedule" },
  { field: "queue", header: "Queue", hideOnMobile: true },
  { field: "last_run_at", header: "Last run" },
]);

onMounted(getData);

async function getData() {
  loading.value = true;
  try {
    jobs.value = await jobsStore.getPeriodic();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col w-full gap-3">
    <div class="flex flex-row justify-between items-center">
      <div class="text-sm" style="color: var(--text-secondary)">
        Jobs the app runs on a fixed schedule. Schedules are defined in code.
      </div>
      <Button class="main-button" style="height: 32px" @click="getData">
        <div class="flex flex-row gap-1 items-center">
          <i class="pi pi-refresh" />
          <span class="mobile-hide"> Refresh </span>
        </div>
      </Button>
    </div>

    <DataTable
      class="w-full enhanced-table"
      data-key="id"
      :loading="loading"
      :value="jobs"
      :row-hover="true"
      :show-gridlines="false"
      scrollable
    >
      <template #empty>
        <div style="padding: 10px">No periodic jobs configured.</div>
      </template>
      <template #loading>
        <LoadingSpinner />
      </template>

      <Column
        v-for="col of activeColumns"
        :key="col.field"
        :header="col.header"
        :field="col.field"
        :header-class="col.hideOnMobile ? 'mobile-hide ' : ''"
        :body-class="col.hideOnMobile ? 'mobile-hide ' : ''"
      >
        <template #body="{ data }">
          <template v-if="col.field === 'last_run_at'">
            {{
              data.last_run_at
                ? dateHelper.formatDate(data.last_run_at, true)
                : "never"
            }}
          </template>
          <template v-else>
            {{ data[col.field] }}
          </template>
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped></style>
