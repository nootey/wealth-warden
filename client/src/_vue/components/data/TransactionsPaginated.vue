<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type { Transaction } from "../../../models/transaction_models.ts";
import type { Column } from "../../../services/filter_registry.ts";
import {computed, onMounted, provide, ref, watch} from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import type {FilterObj} from "../../../models/shared_models.ts";
import FilterMenu from "../filters/FilterMenu.vue";
import ActiveFilters from "../filters/ActiveFilters.vue";
import ActionRow from "../layout/ActionRow.vue";

const props = defineProps<{
  columns: Column[];
  readOnly: boolean;
  accID?: number;
  rows?: number[];
}>();

defineEmits<{
  rowClick: [id: number]
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const { colors } = useChartColors();

const loading = ref(false);
const records = ref<Transaction[]>([]);

const apiPrefix = "transactions";
const includeDeleted = ref(false);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
    account_id: props.accID ?? null,
    include_deleted: includeDeleted.value,
  };
});
const rows = ref([25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value,
});
const page = ref(1);
const sort = ref(filterHelper.initSort("txn_date"));

const filterStorageIndex = ref(apiPrefix + "-filters");
const filters = ref(
  JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"),
);
const filterOverlayRef = ref<any>(null);

onMounted(async () => {
  await getData();
});

watch(includeDeleted, async () => {
  await getData(1); // Reset to page 1 when toggling
});

async function getData(new_page: number | null = null) {
  loading.value = true;
  if (new_page) page.value = new_page;

  try {
    const paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      page.value,
    );

    records.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
  } catch (e) {
    toastStore.errorResponseToast(e)
  }
  finally {
    loading.value = false;
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

async function applyFilters(list: FilterObj[]) {
  if(props.readOnly)
    return;

  filters.value = filterHelper.mergeFilters(filters.value, list);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  await getData();
  filterOverlayRef.value.hide();
}

async function clearFilters() {
  filters.value = [];
  localStorage.removeItem(filterStorageIndex.value);
  cancelFilters();
  await getData();
}

function cancelFilters() {
  filterOverlayRef.value.hide();
}

async function removeFilter(index: number) {
  if (props.readOnly || index < 0 || index >= filters.value.length) return;

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

  await getData();
}

async function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  await getData();
}

function toggleFilterOverlay(event: any) {
  if(props.readOnly)
    return;
  filterOverlayRef.value.toggle(event);
}

function refresh() {
  getData();
}

provide("removeFilter", removeFilter);
provide("switchSort", switchSort);

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
      :columns="props.columns"
      :api-source="apiPrefix"
      @apply="(list) => applyFilters(list)"
      @clear="clearFilters"
      @cancel="cancelFilters"
    />
  </Popover>

  <div class="flex flex-column w-full">

    <div v-if="!readOnly" class="flex flex-row w-full">
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
        <template #includeDeleted>
          <div
            class="flex align-items-center gap-2"
            style="margin-left: auto"
          >
            <span style="font-size: 0.8rem">Include deleted</span>
            <ToggleSwitch v-model="includeDeleted" />
          </div>
        </template>
      </ActionRow>
    </div>

    <DataTable
      data-key="id"
      class="w-full enhanced-table"
      :loading="loading"
      :value="records"
      scrollable
      scroll-height="50vh"
      :row-class="vueHelper.deletedRowClass"
      column-resize-mode="fit"
      scroll-direction="both"
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
            :sortable="!!sort"
            @click="!sort"
          />
        </template>
        <template #body="{ data }">
          <template v-if="col.field === 'amount'">
            <div class="flex flex-row gap-2 align-items-center">
              <i
                class="text-xs"
                :class="
                (data.transaction_type === 'expense'
                  ? data.amount * -1
                  : data.amount) >= 0
                  ? 'pi pi-angle-up'
                  : 'pi pi-angle-down'
              "
                :style="{
                color:
                  (data.transaction_type === 'expense'
                    ? data.amount * -1
                    : data.amount) >= 0
                    ? colors.pos
                    : colors.neg,
              }"
              />
              <span>{{
                  vueHelper.displayAsCurrency(
                    data.transaction_type == "expense"
                      ? data.amount * -1
                      : data.amount,
                  )
                }}</span>
            </div>
          </template>
          <template v-else-if="col.field === 'txn_date'">
            {{ dateHelper.combineDateAndTime(data?.txn_date, data?.created_at) }}
          </template>
          <template v-else-if="col.field === 'account'">
            <div class="flex flex-row gap-2 align-items-center account-row">
    <span class="hover" @click="$emit('rowClick', data.id)">
      {{ data[col.field]?.["name"] }}
    </span>
              <i
                v-if="data[col.field]?.['deleted_at']"
                v-tooltip="'This account is closed.'"
                class="pi pi-ban popup-icon hover-icon"
              />
            </div>
          </template>
          <template v-else-if="col.field === 'category'">
            {{ data[col.field]?.["display_name"] }}
          </template>
          <template v-else-if="col.field === 'description'">
          <span v-tooltip.top="data[col.field]" class="truncate-text">
            {{ data[col.field] }}
          </span>
          </template>
          <template v-else>
            {{ data[col.field] }}
          </template>
        </template>
      </Column>
    </DataTable>
  </div>

</template>

<style scoped>
.hover {
  font-weight: bold;
}
.hover:hover {
  cursor: pointer;
  text-decoration: underline;
}

.account-row .popup-icon {
  opacity: 0;
  transition: opacity 0.15s ease;
}
.account-row:hover .popup-icon {
  opacity: 1;
}

.account-row.advanced .popup-icon {
  opacity: 1;
}
</style>
