<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type { Column } from "../../../services/filter_registry.ts";
import { computed, onMounted, provide, ref, watch } from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import type { InvestmentTrade } from "../../../models/investment_models.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import FilterMenu from "../filters/FilterMenu.vue";
import ActionRow from "../layout/ActionRow.vue";
import ActiveFilters from "../filters/ActiveFilters.vue";
import type {
  FilterObj,
  PaginatorState,
} from "../../../models/shared_models.ts";
import dateHelper from "../../../utils/date_helper.ts";

const props = defineProps<{
  accID?: number;
}>();

defineEmits<{
  updateTrade: [id: number];
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();

const loading = ref(false);
const records = ref<InvestmentTrade[]>([]);

const apiPrefix = "investments/trades";
const includeDeleted = ref(false);
const { colors } = useChartColors();

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
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort("txn_date"));

const filterStorageIndex = ref(apiPrefix + "-filters");
const filters = ref(
  JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"),
);
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
  {
    field: "asset.name",
    header: "Asset",
    hideFromFilter: true,
  },
  {
    field: "asset.ticker",
    header: "Ticker",
    hideOnMobile: true,
  },
  { field: "quantity", header: "Quantity", type: "number" },
  { field: "trade_type", header: "Type" },
  { field: "txn_date", header: "Date", type: "date" },
  {
    field: "value_at_buy",
    header: "Value on buy",
    hideOnMobile: true,
    type: "number",
  },
  {
    field: "current_value",
    header: "Current value",
    hideOnMobile: true,
    type: "number",
  },
  { field: "profit_loss", header: "PNL", type: "number" },
]);

onMounted(async () => {
  await getData();
});

watch(includeDeleted, async () => {
  await getData(1);
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

async function applyFilters(list: FilterObj[]) {
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
      :columns="activeColumns"
      api-source="investment_trades"
      @apply="(list) => applyFilters(list)"
      @clear="clearFilters"
      @cancel="cancelFilters"
    />
  </Popover>

  <div class="flex flex-column w-full">
    <div class="flex flex-row w-full">
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
        v-for="col of activeColumns"
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
          <template v-if="col.field === 'average_buy_price'">
            <div class="flex flex-row gap-2 align-items-center">
              <span>{{
                vueHelper.displayAssetPrice(
                  data.average_buy_price,
                  data.investment_type,
                )
              }}</span>
            </div>
          </template>
          <template
            v-else-if="
              ['current_price', 'value_at_buy', 'current_value'].includes(
                col.field,
              )
            "
          >
            <div class="flex flex-row gap-2 align-items-center">
              <span>{{
                vueHelper.displayAsCurrency(data[col.field], data["currency"])
              }}</span>
            </div>
          </template>
          <template v-else-if="col.field === 'txn_date'">
            {{
              dateHelper.combineDateAndTime(data?.txn_date, data?.created_at)
            }}
          </template>
          <template v-else-if="col.field == 'profit_loss'">
            <div class="flex flex-row gap-2 align-items-center">
              <i
                class="text-xs"
                :class="
                  data[col.field] >= 0 ? 'pi pi-angle-up' : 'pi pi-angle-down'
                "
                :style="{
                  color: data[col.field] >= 0 ? colors.pos : colors.neg,
                }"
              />
              <span>
                {{
                  vueHelper.displayAsCurrency(data[col.field], data["currency"])
                }}
              </span>
            </div>
          </template>
          <template
            v-else-if="
              col.field === 'asset.name' || col.field === 'asset.ticker'
            "
          >
            <div class="flex flex-row gap-2 align-items-center">
              <span
                :class="{ hover: col.field === 'asset.name' }"
                @click="
                  col.field === 'asset.name'
                    ? $emit('updateTrade', data.id)
                    : null
                "
              >
                {{
                  col.field === "asset.name"
                    ? data.asset.name
                    : data.asset.ticker
                }}
              </span>
            </div>
          </template>
          <template v-else-if="col.field === 'trade_type'">
            <div class="flex flex-row gap-2 align-items-center">
              <span
                :style="{
                  color: data[col.field] === 'buy' ? colors.pos : colors.neg,
                }"
              >
                {{
                  data[col.field].charAt(0).toUpperCase() +
                  data[col.field].slice(1)
                }}
              </span>
            </div>
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
</style>
