<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import { useJobsStore } from "../../services/stores/jobs_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useWsStore } from "../../services/stores/ws_store.ts";
import dateHelper from "../../utils/date_helper.ts";
import type { Column } from "../../services/filter_registry.ts";
import type { JobState, RiverJob } from "../../models/job_models.ts";
import type { UserJobPayload } from "../../models/ws_models.ts";

const props = defineProps<{
  kind: string;
  label?: string;
  description?: string;
}>();

const jobsStore = useJobsStore();
const toastStore = useToastStore();
const wsStore = useWsStore();
const confirm = useConfirm();

const jobs = ref<RiverJob[]>([]);
const loading = ref(false);

// Job args are keyed by field names. UserID and anything named Internal* is excluded - worker only.
const INTERNAL_ARG_PREFIX = "Internal";

const ACTIVE_STATES: JobState[] = [
  "available",
  "scheduled",
  "pending",
  "retryable",
  "running",
];

const RETRYABLE_STATES: JobState[] = ["discarded", "cancelled", "retryable"];

const activeColumns = computed<Column[]>(() => [
  { field: "id", header: "ID", hideOnMobile: true },
  { field: "state", header: "State" },
  { field: "attempt", header: "Attempt" },
  { field: "args", header: "Args" },
  { field: "created_at", header: "Created", hideOnMobile: true },
]);

const stateSeverity: Record<string, string> = {
  running: "info",
  completed: "success",
  discarded: "danger",
  retryable: "warn",
  cancelled: "secondary",
  available: "contrast",
  scheduled: "contrast",
  pending: "contrast",
};



function humanizeKey(key: string): string {
  return key
    .replace(/_/g, " ")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .split(" ")
    .map((word, index) =>
      word === word.toUpperCase() || index === 0 ? word : word.toLowerCase(),
    )
    .join(" ");
}

function formatArgValue(value: unknown): string {
  if (value === null || value === undefined) return "-";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function argEntries(job: RiverJob): { label: string; value: string }[] {
  if (!job.args || typeof job.args !== "object") return [];
  return Object.entries(job.args as Record<string, unknown>)
    .filter(([key]) => key !== "UserID" && !key.startsWith(INTERNAL_ARG_PREFIX))
    .map(([key, value]) => ({
      label: humanizeKey(key),
      value: formatArgValue(value),
    }));
}

function canRetry(job: RiverJob): boolean {
  return RETRYABLE_STATES.includes(job.state);
}

function canCancel(job: RiverJob): boolean {
  return ACTIVE_STATES.includes(job.state);
}

let unsubscribe: (() => void) | null = null;

onMounted(async () => {
  await refresh();
  unsubscribe = wsStore.on("user_job.updated", (payload) => {
    if ((payload as UserJobPayload | undefined)?.kind === props.kind) {
      void refresh(true);
    }
  });
});

onBeforeUnmount(() => {
  unsubscribe?.();
});

async function refresh(silent = false) {
  if (!silent) loading.value = true;
  try {
    jobs.value = await jobsStore.listJobs(props.kind);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

function retry(job: RiverJob) {
  confirm.require({
    header: `Retry job #${job.id}?`,
    message: `This will retry the ${props.label ?? job.kind} job.`,
    rejectProps: { label: "Cancel", severity: "secondary" },
    acceptProps: { label: "Retry" },
    accept: async () => {
      try {
        const res = await jobsStore.retryJob(job.id);
        toastStore.successResponseToast(res);
        await refresh(true);
      } catch (error) {
        toastStore.errorResponseToast(error);
      }
    },
  });
}

function cancel(job: RiverJob) {
  confirm.require({
    header: `Cancel job #${job.id}?`,
    message: `This will cancel the ${props.label ?? job.kind} job.`,
    rejectProps: { label: "Back", severity: "secondary" },
    acceptProps: { label: "Cancel job", severity: "danger" },
    accept: async () => {
      try {
        const res = await jobsStore.cancelJob(job.id);
        toastStore.successResponseToast(res);
        await refresh(true);
      } catch (error) {
        toastStore.errorResponseToast(error);
      }
    },
  });
}
</script>

<template>
  <div class="flex flex-col gap-2 p-4 border rounded-md border-surface">
    <div v-if="label" class="font-bold">{{ label }}</div>
    <div v-if="description" class="text-sm text-muted-color">
      {{ description }}
    </div>

    <DataTable
      class="w-full"
      data-key="id"
      :loading="loading"
      :value="jobs"
      :row-hover="true"
      :show-gridlines="false"
    >
      <template #empty>
        <div class="p-2 text-sm text-muted-color">No jobs run yet.</div>
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
          <template v-if="col.field === 'state'">
            <Tag
              :value="data.state"
              :severity="stateSeverity[data.state] ?? 'secondary'"
            />
          </template>
          <template v-else-if="col.field === 'attempt'">
            {{ data.attempt }}/{{ data.max_attempts }}
          </template>
          <template v-else-if="col.field === 'args'">
            <div
              v-if="argEntries(data).length"
              class="flex flex-col gap-1 max-w-64"
            >
              <span
                v-for="entry in argEntries(data)"
                :key="entry.label"
                class="text-xs truncate"
                :title="entry.value"
              >
                <span class="text-muted-color">{{ entry.label }}:</span>
                {{ entry.value }}
              </span>
            </div>
            <span v-else class="text-xs text-muted-color">-</span>
          </template>
          <template v-else-if="col.field === 'created_at'">
            {{ dateHelper.formatDate(data.created_at, true) }}
          </template>
          <template v-else>
            {{ data[col.field] }}
          </template>
        </template>
      </Column>
      <Column header="Actions">
        <template #body="{ data }">
          <div class="flex flex-row gap-2">
            <Button
              v-if="canRetry(data)"
              size="small"
              class="outline-button"
              icon="pi pi-replay"
              @click="retry(data)"
            />
            <Button
              v-if="canCancel(data)"
              size="small"
              class="delete-button"
              icon="pi pi-ban"
              @click="cancel(data)"
            />
          </div>
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped></style>
