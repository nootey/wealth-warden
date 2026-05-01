<script setup lang="ts">
import { onMounted, computed, ref } from "vue";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import dateHelper from "../../../utils/date_helper.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import DisplayStatus from "../base/DisplayStatus.vue";
import type { PaginatorState } from "../../../models/shared_models.ts";
import type { Report } from "../../../models/analytics_models.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();

const apiPrefix = "analytics/reports";

const loading = ref(false);
const records = ref<Report[]>([]);

const rows = ref([10, 25, 50]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: rows.value[0]!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort("created_at"));

const params = computed(() => ({
  rowsPerPage: paginator.value.rowsPerPage,
  sort: sort.value,
}));

onMounted(async () => {
  await getData();
});

async function getData(newPage: number | null = null) {
  loading.value = true;
  if (newPage) page.value = newPage;

  try {
    const res = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      page.value,
    );
    records.value = res.data.records;
    paginator.value.total = res.total_records;
    paginator.value.from = res.from;
    paginator.value.to = res.to;
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

function refresh() {
  getData(1);
}

defineExpose({ refresh });
</script>

<template>
  <div class="flex flex-column w-full gap-3">
    <div
      class="flex flex-column w-full border-round-2xl"
      style="
        padding: 0.25rem 0.25rem 0 0.25rem;
        border: 1px solid var(--border-color);
      "
    >
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

        <Column field="name" header="Name" />
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
