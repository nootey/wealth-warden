<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import { useJobsStore } from "../../../services/stores/jobs_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import type { Column } from "../../../services/filter_registry.ts";
import type { RiverQueue } from "../../../models/job_models.ts";

const jobsStore = useJobsStore();
const toastStore = useToastStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const canManage = computed(() => hasPermission("manage_jobs"));

const loading = ref(false);
const queues = ref<RiverQueue[]>([]);

const activeColumns = computed<Column[]>(() => [
  { field: "name", header: "Queue" },
  { field: "status", header: "Status" },
  { field: "jobs", header: "Jobs" },
]);

onMounted(getData);

async function getData() {
  loading.value = true;
  try {
    queues.value = await jobsStore.getQueues();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

function countsSummary(q: RiverQueue): string {
  if (!q.counts) return "-";
  const parts = Object.entries(q.counts)
    .filter(([, n]) => n > 0)
    .map(([state, n]) => `${state}: ${n}`);
  return parts.length > 0 ? parts.join(", ") : "-";
}

function confirmToggle(q: RiverQueue) {
  const paused = q.paused_at != null;
  const verb = paused ? "Resume" : "Pause";
  confirm.require({
    header: `${verb} queue "${q.name}"?`,
    message: paused
      ? "Workers will start fetching jobs from this queue again."
      : "Workers will stop fetching new jobs from this queue. Running jobs finish.",
    rejectProps: { label: "Cancel", severity: "secondary" },
    acceptProps: { label: verb },
    accept: async () => {
      try {
        const res = paused
          ? await jobsStore.resumeQueue(q.name)
          : await jobsStore.pauseQueue(q.name);
        toastStore.successResponseToast(res);
        await getData();
      } catch (error) {
        toastStore.errorResponseToast(error);
      }
    },
  });
}
</script>

<template>
  <div class="flex flex-col w-full gap-3">
    <div class="flex flex-row justify-between items-center gap-2">
      <div class="text-sm" style="color: var(--text-secondary)">
        Worker pools that pull jobs from the database. Pausing a queue stops
        workers from picking up new jobs.
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
      data-key="name"
      :loading="loading"
      :value="queues"
      :row-hover="true"
      :show-gridlines="false"
      scrollable
    >
      <template #empty>
        <div style="padding: 10px">No active queues.</div>
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
          <template v-if="col.field === 'status'">
            <Tag v-if="data.paused_at" value="paused" severity="warn" />
            <Tag v-else value="active" severity="success" />
          </template>
          <template v-else-if="col.field === 'jobs'">
            {{ countsSummary(data) }}
          </template>
          <template v-else>
            {{ data[col.field] }}
          </template>
        </template>
      </Column>
      <Column v-if="canManage">
        <template #header>
          <span class="mobile-hide">Actions</span>
        </template>
        <template #body="{ data }">
          <i
            v-tooltip.top="data.paused_at ? 'Resume queue' : 'Pause queue'"
            class="hover-icon"
            :class="data.paused_at ? 'pi pi-play' : 'pi pi-pause'"
            style="font-size: 0.875rem"
            @click="confirmToggle(data)"
          />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped></style>
