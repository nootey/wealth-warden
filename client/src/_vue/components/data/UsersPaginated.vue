<script setup lang="ts">
import ActiveFilters from "../filters/ActiveFilters.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import type { Column } from "../../../services/filter_registry.ts";
import ActionRow from "../layout/ActionRow.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import CustomPaginator from "../base/CustomPaginator.vue";
import type {
  FilterObj,
  PaginatorState,
} from "../../../models/shared_models.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import { computed, onMounted, provide, ref } from "vue";
import type { Role, User } from "../../../models/user_models.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useUserStore } from "../../../services/stores/user_store.ts";
import FilterMenu from "../filters/FilterMenu.vue";
import dateHelper from "../../../utils/date_helper.ts";

const props = defineProps<{
  roles: Role[];
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "updateUser", value: number): void;
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const userStore = useUserStore();

const apiPrefix = userStore.apiPrefix;

const loading = ref(false);
const records = ref<User[]>([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
  };
});

const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort());
const filterStorageIndex = ref(apiPrefix + "-filters");
const filters = ref(
  JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"),
);
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
  { field: "display_name", header: "Name", type: "text" },
  { field: "email", header: "Email", type: "text" },
  {
    field: "role",
    header: "Role",
    type: "enum",
    options: props.roles,
    optionLabel: "name",
  },
  { field: "email_confirmed", header: "Verified", type: "date" },
]);

onMounted(async () => {
  await init();
});

async function init() {
  await getData();
}

async function getData(new_page: number | null = null) {
  loading.value = true;
  if (new_page) page.value = new_page;

  try {
    let payload = {
      ...params.value,
    };

    let paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      payload,
      page.value,
    );

    records.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loading.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

function applyFilters(list: FilterObj[]) {
  filters.value = filterHelper.mergeFilters(filters.value, list);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  getData();
  filterOverlayRef.value.hide();
}

function clearFilters() {
  filters.value = [];
  localStorage.removeItem(filterStorageIndex.value);
  cancelFilters();
  getData();
}

function cancelFilters() {
  filterOverlayRef.value.hide();
}

function removeFilter(index: number) {
  if (index < 0 || index >= filters.value.length) return;

  const next = filters.value.slice();
  next.splice(index, 1);
  filters.value = next;

  if (filters.value.length > 0) {
    localStorage.setItem(
      filterStorageIndex.value,
      JSON.stringify(filters.value),
    );
  } else {
    localStorage.removeItem(filterStorageIndex.value);
  }

  getData();
}

function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
}

function toggleFilterOverlay(event: any) {
  filterOverlayRef.value.toggle(event);
}

function refresh() {
  getData();
}

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);

defineExpose({ refresh });
</script>

<template>
  <Popover
    ref="filterOverlayRef"
    class="rounded-popover"
    :style="{ width: '420px' }"
    :breakpoints="{ '775px': '90vw' }"
  >
    <FilterMenu
      v-model:value="filters"
      :columns="activeColumns"
      :api-source="apiPrefix"
      @apply="(list) => applyFilters(list)"
      @clear="clearFilters"
      @cancel="cancelFilters"
    />
  </Popover>

  <div class="flex flex-column w-full gap-3">
    <div
      class="flex flex-row justify-content-between align-items-center p-1 gap-3 w-full border-round-md"
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
            class="hover-icon flex flex-row align-items-center gap-2"
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

    <div id="mobile-row" class="flex flex-row gap-2 w-full">
      <DataTable
        class="w-full enhanced-table"
        data-key="id"
        :loading="loading"
        :value="records"
        :row-hover="true"
        :show-gridlines="false"
        scrollable
        column-resize-mode="fit"
      >
        <template #empty>
          <div style="padding: 10px">No records found.</div>
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
          v-for="col of activeColumns"
          :key="col.field"
          :field="col.field"
        >
          <template #header>
            <ColumnHeader
              :header="col.header"
              :field="col.field"
              :sort="sort"
            />
          </template>
          <template #body="{ data }">
            <template v-if="col.field === 'email_confirmed'">
              {{ dateHelper.formatDate(data?.email_confirmed, true) }}
            </template>
            <template v-else-if="col.field === 'display_name'">
              <span
                class="hover-icon font-bold"
                @click="emit('updateUser', data.id)"
              >
                {{ data[col.field] }}
              </span>
            </template>
            <template v-else-if="col.field === 'role'">
              <span>
                {{ data[col.field]["name"] }}
              </span>
            </template>
            <template v-else>
              {{ data[col.field] }}
            </template>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped></style>
