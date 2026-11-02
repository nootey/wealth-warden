<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref } from "vue";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import { useWsStore } from "../../../services/stores/ws_store.ts";
import dateHelper from "../../../utils/date_helper.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import { useCursorPager } from "../../../utils/use_cursor_pager.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import DisplayStatus from "../base/DisplayStatus.vue";
import type { Report } from "../../../models/analytics_models.ts";
import { useConfirm } from "primevue/useconfirm";
import vueHelper from "../../../utils/vue_helper.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const analyticsStore = useAnalyticsStore();
const wsStore = useWsStore();

const apiPrefix = "analytics/reports";

const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const records = ref<Report[]>([]);

const rows = ref([10, 25, 50]);
const pager = useCursorPager(rows.value[0]!);
const paginator = pager.paginator;
const sort = ref(filterHelper.initSort("created_at"));

const params = computed(() => ({
  rowsPerPage: paginator.value.rowsPerPage,
  sort: sort.value,
}));

const unsubscribers: (() => void)[] = [];

onMounted(async () => {
  await getData();

  unsubscribers.push(
    wsStore.on("report.completed", () => {
      toastStore.createInfoToast("Report ready", "Your report has finished.");
      refresh();
    }),
    wsStore.on("report.failed", () => refresh()),
  );
});

onUnmounted(() => unsubscribers.forEach((unsubscribe) => unsubscribe()));

async function getData() {
  loading.value = true;

  try {
    const res = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      pager.cursor.value,
    );
    records.value = res.data.records;
    pager.update(res, records.value.length);
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function onPage(event: any) {
  if (pager.goTo(event.page, event.rows)) await getData();
}

async function downloadReport(id: number, name: string) {
  try {
    await analyticsStore.downloadReport(id, name);
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function deleteConfirmation(id: number, name: string) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete report: "${id} - ${vueHelper.truncate(name)}".`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteReport(id),
  });
}

async function deleteReport(id: number) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord(apiPrefix, id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

function refresh() {
  pager.reset();
  getData();
}

defineExpose({ refresh });
</script>

<template>
  <div class="flex flex-col w-full gap-4">
    <div class="flex flex-col w-full">
      <DataTable
        class="w-full enhanced-table"
        data-key="id"
        :loading="loading"
        :value="records"
        size="small"
      >
        <template #empty>
          <div style="padding: 10px">No reports generated yet.</div>
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

        <Column header="Actions">
          <template #body="{ data }">
            <div class="flex flex-row gap-2 items-center">
              <i
                v-if="data.status === 'completed'"
                class="pi pi-download hover-icon text-sm"
                @click="downloadReport(data.id, data.name)"
              />
              <i
                v-if="data.status === 'completed' || data.status === 'failed'"
                class="pi pi-trash hover-icon text-sm"
                style="color: var(--p-red-300)"
                @click="deleteConfirmation(data.id, data.name)"
              />
            </div>
          </template>
        </Column>
        <Column field="name" header="Name">
          <template #body="{ data }">
            <span
              v-tooltip.top="data.name"
              class="block truncate max-w-[9rem] sm:max-w-[16rem] md:max-w-[30rem]"
            >
              {{ data.name }}
            </span>
          </template>
        </Column>
        <Column field="type" header="Type" />
        <Column field="status" header="Status">
          <template #body="{ data }">
            <DisplayStatus :status="data.status" />
          </template>
        </Column>
        <Column field="created_at" header="Generated">
          <template #body="{ data }">
            {{ dateHelper.formatDate(data.created_at, true) }}
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>
