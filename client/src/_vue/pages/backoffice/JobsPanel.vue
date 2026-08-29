<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, provide, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import { useJobsStore } from "../../../services/stores/jobs_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import dateHelper from "../../../utils/date_helper.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import CustomPaginator from "../../components/base/CustomPaginator.vue";
import ColumnHeader from "../../components/base/ColumnHeader.vue";
import FilterMenu from "../../components/filters/FilterMenu.vue";
import ActiveFilters from "../../components/filters/ActiveFilters.vue";
import { JOB_KINDS, JOB_STATES } from "../../../models/job_models.ts";
import type { Column } from "../../../services/filter_registry.ts";
import type {
  JobCounts,
  JobState,
  RiverJob,
  RiverJobDetail,
} from "../../../models/job_models.ts";
import type {
  FilterObj,
  PaginatorState,
} from "../../../models/shared_models.ts";
import ActionRow from "../../components/layout/ActionRow.vue";

const jobsStore = useJobsStore();
const toastStore = useToastStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const POLL_MS = 4000;

const loading = ref(false);
const records = ref<RiverJob[]>([]);
const counts = ref<JobCounts>({} as JobCounts);

const FILTER_STORAGE_KEY = "backoffice-jobs-filters";

const selectedStates = ref<JobState[]>([]);
const filters = ref<FilterObj[]>(
  JSON.parse(localStorage.getItem(FILTER_STORAGE_KEY) ?? "[]"),
);
const filterOverlayRef = ref<{
  toggle: (e: Event) => void;
  hide: () => void;
}>();

const queueOptions = ref<{ label: string; value: string }[]>([]);
const kindOptions = JOB_KINDS.map((k) => ({ label: k, value: k }));

const columns = computed<Column[]>(() => [
  {
    field: "id",
    header: "ID",
    type: "int",
    sortable: true,
    hideOnMobile: true,
  },
  {
    field: "kind",
    header: "Name",
    type: "enum",
    options: kindOptions,
    optionLabel: "label",
    optionValue: "value",
    sortable: true,
  },
  {
    field: "queue",
    header: "Queue",
    type: "enum",
    options: queueOptions.value,
    optionLabel: "label",
    optionValue: "value",
    sortable: true,
    hideOnMobile: true,
  },
  { field: "state", header: "State", sortable: true, hideFromFilter: true },
  {
    field: "attempt",
    header: "Attempt",
    sortable: true,
    hideFromFilter: true,
    hideOnMobile: true,
  },
  {
    field: "created_at",
    header: "Created",
    type: "date",
    sortable: true,
    hideFromFilter: true,
    hideOnMobile: true,
  },
  {
    field: "scheduled_at",
    header: "Scheduled",
    type: "date",
    sortable: true,
    hideFromFilter: true,
    hideOnMobile: true,
  },
  {
    field: "duration",
    header: "Duration",
    sortable: true,
    hideFromFilter: true,
  },
]);

const rows = ref([10, 25, 50, 100]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: rows.value[0]!,
});
const page = ref(1);
const sort = ref({ field: "id", order: -1 });

const detail = ref<RiverJobDetail | null>(null);
const detailVisible = ref(false);
const loadingDetail = ref(false);

const RETRYABLE_STATES: JobState[] = ["discarded", "cancelled", "retryable"];
const CANCELLABLE_STATES: JobState[] = [
  "available",
  "scheduled",
  "pending",
  "retryable",
  "running",
];

const canRetry = computed(
  () => !!detail.value && RETRYABLE_STATES.includes(detail.value.state),
);
const canCancel = computed(
  () => !!detail.value && CANCELLABLE_STATES.includes(detail.value.state),
);

let pollTimer: number | undefined;

const params = computed(() => ({
  rowsPerPage: paginator.value.rowsPerPage,
  sort: sort.value,
  states: selectedStates.value,
  filters: filters.value,
}));

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

onMounted(async () => {
  await Promise.all([getData(), getCounts(), getQueues()]);
  startPolling();
  document.addEventListener("visibilitychange", onVisibilityChange);
});

onBeforeUnmount(() => {
  stopPolling();
  document.removeEventListener("visibilitychange", onVisibilityChange);
});

function startPolling() {
  stopPolling();
  pollTimer = window.setInterval(() => {
    if (document.hidden) return;
    void getCounts();
    void getData(null, true);
  }, POLL_MS);
}

function stopPolling() {
  if (pollTimer) window.clearInterval(pollTimer);
  pollTimer = undefined;
}

function onVisibilityChange() {
  if (!document.hidden) {
    void getCounts();
    void getData(null, true);
  }
}

async function getData(newPage: number | null = null, silent = false) {
  if (newPage) page.value = newPage;
  if (!silent) loading.value = true;

  try {
    const response = await jobsStore.getJobsPaginated(params.value, page.value);
    records.value = response.data ?? [];
    paginator.value.total = response.total_records;
    paginator.value.to = response.to;
    paginator.value.from = response.from;
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function getCounts() {
  try {
    counts.value = await jobsStore.getCounts();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getQueues() {
  try {
    const queues = await jobsStore.getQueues();
    queueOptions.value = queues.map((q) => ({ label: q.name, value: q.name }));
  } catch {
    // queues panel surfaces its own errors
  }
}

function toggleState(state: JobState) {
  const idx = selectedStates.value.indexOf(state);
  if (idx === -1) selectedStates.value.push(state);
  else selectedStates.value.splice(idx, 1);
  getData(1);
}

function applyFilters(list: FilterObj[]) {
  filters.value = filterHelper.mergeFilters(filters.value, list);
  persistFilters();
  filterOverlayRef.value?.hide();
  getData(1);
}

function clearFilters() {
  filters.value = [];
  persistFilters();
  filterOverlayRef.value?.hide();
  getData(1);
}

function removeFilter(index: number) {
  if (index < 0 || index >= filters.value.length) return;
  const next = filters.value.slice();
  next.splice(index, 1);
  filters.value = next;
  persistFilters();
  getData(1);
}

function persistFilters() {
  if (filters.value.length > 0) {
    localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(filters.value));
  } else {
    localStorage.removeItem(FILTER_STORAGE_KEY);
  }
}

function toggleFilterOverlay(event: Event) {
  filterOverlayRef.value?.toggle(event);
}

provide("removeFilter", removeFilter);

async function onPage(event: { rows: number; page: number }) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

function switchSort(field: string) {
  if (sort.value.field === field) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.field = field;
    sort.value.order = -1;
  }
  getData();
}

provide("switchSort", switchSort);

async function openDetail(id: number) {
  detailVisible.value = true;
  loadingDetail.value = true;
  detail.value = null;
  try {
    detail.value = await jobsStore.getJob(id);
  } catch (error) {
    toastStore.errorResponseToast(error);
    detailVisible.value = false;
  } finally {
    loadingDetail.value = false;
  }
}

function confirmAction(
  verb: string,
  id: number,
  action: (ids: number[]) => Promise<unknown>,
) {
  confirm.require({
    header: `${verb} job #${id}?`,
    message: `This will ${verb.toLowerCase()} job #${id}.`,
    rejectProps: { label: "Cancel", severity: "secondary" },
    acceptProps: {
      label: verb,
      severity: verb === "Delete" ? "danger" : undefined,
    },
    accept: async () => {
      try {
        const res = await action([id]);
        toastStore.successResponseToast(res!);
        await Promise.all([getData(null, true), getCounts()]);
      } catch (error) {
        toastStore.errorResponseToast(error);
      }
    },
  });
}

function duration(job: RiverJob): string {
  if (!job.attempted_at) return "-";
  const end = job.finalized_at ?? new Date().toISOString();
  const ms = new Date(end).getTime() - new Date(job.attempted_at).getTime();
  if (ms < 0) return "-";
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)} s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

function prettyJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
</script>

<template>
  <div>
    <div class="flex flex-col w-full gap-4">
      <div class="text-sm" style="color: var(--text-secondary)">
        Every background job the app has queued or run. Finished jobs
        life-cycle: 7 days - completed and cancelled, 30 days - discarded
      </div>

      <div class="text-sm" style="color: var(--text-secondary)">
        A job that runs past its time limit (15 minutes by default) is stopped
        and retried, and moves to discarded once it runs out of attempts.
      </div>

      <div class="flex flex-row flex-wrap gap-2">
        <button
          v-for="state in JOB_STATES"
          :key="state"
          type="button"
          class="flex flex-row items-center gap-2 px-3 py-1 rounded-md text-sm"
          :style="{
            border: '1px solid var(--border-color)',
            background: selectedStates.includes(state)
              ? 'var(--text-primary)'
              : 'var(--background-secondary)',
            color: selectedStates.includes(state)
              ? 'var(--background-primary)'
              : 'var(--text-secondary)',
          }"
          @click="toggleState(state)"
        >
          <span>{{ state }}</span>
          <span class="font-bold">{{ counts[state] ?? 0 }}</span>
        </button>
      </div>

      <div
        class="flex flex-row flex-wrap items-center gap-3 p-2 rounded-md"
        style="
          border: 1px solid var(--border-color);
          background: var(--background-secondary);
        "
      >
        <ActionRow>
          <template #activeFilters>
            <ActiveFilters
              :active-filters="filters"
              :show-only-active="false"
              active-filter=""
            />
          </template>
          <template #filterButton>
            <div
              class="hover-icon flex flex-row items-center gap-2"
              style="
                padding: 0.5rem 1rem;
                border-radius: 8px;
                border: 1px solid var(--border-color);
              "
              @click="toggleFilterOverlay($event)"
            >
              <i class="pi pi-filter" style="font-size: 0.845rem" />
              <div>Filter</div>
            </div>
          </template>
        </ActionRow>
      </div>

      <Popover
        ref="filterOverlayRef"
        class="rounded-popover"
        :style="{ width: '420px' }"
        :breakpoints="{ '775px': '90vw' }"
      >
        <FilterMenu
          v-model:value="filters"
          :columns="columns"
          api-source="jobs"
          @apply="applyFilters"
          @clear="clearFilters"
          @cancel="filterOverlayRef?.hide()"
        />
      </Popover>

      <DataTable
        class="w-full enhanced-table"
        data-key="id"
        :loading="loading"
        :value="records"
        :row-hover="true"
        :show-gridlines="false"
        scrollable
      >
        <template #empty>
          <div style="padding: 10px">No jobs found.</div>
        </template>
        <template #loading>
          <LoadingSpinner />
        </template>
        <template #footer>
          <CustomPaginator
            :paginator="paginator"
            :rows="rows"
            @on-page="onPage"
          />
        </template>

        <Column
          v-for="col of columns"
          :key="col.field"
          :field="col.field"
          :header-class="col.hideOnMobile ? 'mobile-hide ' : ''"
          :body-class="col.hideOnMobile ? 'mobile-hide ' : ''"
        >
          <template #header>
            <ColumnHeader
              :header="col.header"
              :field="col.field"
              :sort="sort"
              :sortable="col.sortable"
            />
          </template>
          <template #body="{ data }">
            <template v-if="col.field === 'kind'">
              <span class="hover-icon font-bold" @click="openDetail(data.id)">
                {{ data.kind }}
              </span>
            </template>
            <template v-else-if="col.field === 'state'">
              <Tag
                :value="data.state"
                :severity="stateSeverity[data.state] ?? 'secondary'"
              />
            </template>
            <template v-else-if="col.field === 'attempt'">
              {{ data.attempt }}/{{ data.max_attempts }}
            </template>
            <template v-else-if="col.field === 'created_at'">
              {{ dateHelper.formatDate(data.created_at, true) }}
            </template>
            <template v-else-if="col.field === 'scheduled_at'">
              {{ dateHelper.formatDate(data.scheduled_at, true) }}
            </template>
            <template v-else-if="col.field === 'duration'">
              {{ duration(data) }}
            </template>
            <template v-else>
              {{ data[col.field] }}
            </template>
          </template>
        </Column>
      </DataTable>
    </div>

    <Dialog
      v-model:visible="detailVisible"
      position="right"
      class="rounded-dialog"
      :modal="true"
      :breakpoints="{ '701px': '95vw' }"
      :style="{ width: '640px' }"
      :header="detail ? `Job #${detail.id} - ${detail.kind}` : 'Job'"
    >
      <div v-if="loadingDetail" class="p-4">
        <LoadingSpinner />
      </div>
      <div v-else-if="detail" class="flex flex-col gap-4 p-1 text-sm">
        <div class="flex flex-row flex-wrap gap-x-6 gap-y-1">
          <div>
            <span class="text-muted-color">State: </span>
            <Tag
              :value="detail.state"
              :severity="stateSeverity[detail.state] ?? 'secondary'"
            />
          </div>
          <div>
            <span class="text-muted-color">Queue: </span>{{ detail.queue }}
          </div>
          <div>
            <span class="text-muted-color">Attempt: </span
            >{{ detail.attempt }}/{{ detail.max_attempts }}
          </div>
          <div>
            <span class="text-muted-color">Priority: </span
            >{{ detail.priority }}
          </div>
        </div>

        <div class="flex flex-col gap-1">
          <div>
            <span class="text-muted-color">Created: </span
            >{{ dateHelper.formatDate(detail.created_at, true) }}
          </div>
          <div>
            <span class="text-muted-color">Scheduled: </span
            >{{ dateHelper.formatDate(detail.scheduled_at, true) }}
          </div>
          <div v-if="detail.attempted_at">
            <span class="text-muted-color">Last attempt: </span
            >{{ dateHelper.formatDate(detail.attempted_at, true) }}
          </div>
          <div v-if="detail.finalized_at">
            <span class="text-muted-color">Finalized: </span
            >{{ dateHelper.formatDate(detail.finalized_at, true) }}
          </div>
        </div>

        <div>
          <div class="font-bold mb-1">Args</div>
          <pre
            class="p-2 rounded-md overflow-x-auto"
            style="background: var(--background-secondary)"
            >{{ prettyJson(detail.args) }}</pre>
        </div>

        <div>
          <div class="font-bold mb-1">Metadata</div>
          <pre
            class="p-2 rounded-md overflow-x-auto"
            style="background: var(--background-secondary)"
            >{{ prettyJson(detail.metadata) }}</pre>
        </div>

        <div v-if="detail.errors && detail.errors.length > 0">
          <div class="font-bold mb-1">Errors ({{ detail.errors.length }})</div>
          <div class="flex flex-col gap-2">
            <div
              v-for="(err, i) in detail.errors"
              :key="i"
              class="p-2 rounded-md"
              style="border: 1px solid var(--border-color)"
            >
              <div class="text-muted-color">
                Attempt {{ err.attempt }} -
                {{ dateHelper.formatDate(err.at, true) }}
              </div>
              <div class="whitespace-pre-wrap break-words">{{ err.error }}</div>
              <details v-if="err.trace" class="mt-1">
                <summary class="cursor-pointer text-muted-color">
                  Stack trace
                </summary>
                <pre class="overflow-x-auto text-xs">{{ err.trace }}</pre>
              </details>
            </div>
          </div>
        </div>

        <div
          v-if="hasPermission('manage_jobs')"
          class="flex flex-row gap-2 pt-2"
          style="border-top: 1px solid var(--border-color)"
        >
          <Button
            v-if="canRetry"
            label="Retry"
            size="small"
            class="main-button"
            icon="pi pi-replay"
            @click="confirmAction('Retry', detail.id, jobsStore.retryJobs)"
          />
          <Button
            v-if="canCancel"
            label="Cancel"
            size="small"
            class="outline-button"
            icon="pi pi-ban"
            @click="confirmAction('Cancel', detail.id, jobsStore.cancelJobs)"
          />
          <Button
            label="Delete"
            size="small"
            class="delete-button"
            icon="pi pi-trash"
            @click="confirmAction('Delete', detail.id, jobsStore.deleteJobs)"
          />
        </div>
      </div>
    </Dialog>
  </div>
</template>

<style scoped></style>
