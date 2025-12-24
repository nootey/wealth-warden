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
import type { InvestmentTransaction } from "../../../models/investment_models.ts";
import {useChartColors} from "../../../style/theme/chartColors.ts";

const props = defineProps<{
  accID?: number;
}>();

defineEmits<{
  updateTransaction: [id: number];
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();

const loading = ref(false);
const records = ref<InvestmentTransaction[]>([]);

const apiPrefix = "investments/transactions";
const includeDeleted = ref(false);
const { colors } = useChartColors();

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: [],
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
const sort = ref(filterHelper.initSort());

const activeColumns = computed<Column[]>(() => [
  {
    field: "holding",
    header: "Asset",
    type: "enum",
    optionLabel: "name",
  },
  {
    field: "holding",
    header: "Ticker",
    type: "enum",
    optionLabel: "ticker",
    hideOnMobile: true
  },
  { field: "quantity", header: "Quantity" },
  { field: "transaction_type", header: "Type"},
  { field: "value_at_buy", header: "Value on buy", hideOnMobile: true },
  { field: "current_value", header: "Current value", hideOnMobile: true },
  { field: "profit_loss", header: "PNL" },

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

async function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  await getData();
}

function refresh() {
  getData();
}

provide("switchSort", switchSort);

defineExpose({ refresh });
</script>

<template>
  <div class="flex flex-column w-full">
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
          <template v-else-if="['current_price', 'value_at_buy', 'current_value'].includes(col.field)">
            <div class="flex flex-row gap-2 align-items-center">
              <span>{{ vueHelper.displayAsCurrency(data[col.field]) }}</span>
            </div>
          </template>
          <template v-else-if="col.field == 'profit_loss'">
            <div class="flex flex-row gap-2 align-items-center">
              <i
                class="text-xs"
                :class="data[col.field] >= 0 ? 'pi pi-angle-up' : 'pi pi-angle-down'"
                :style="{ color: data[col.field] >= 0 ? colors.pos : colors.neg }"
              />
              <span>
                {{ vueHelper.displayAsCurrency(data[col.field]) }}
              </span>
            </div>
          </template>
          <template v-else-if="col.field === 'holding'">
            <div class="flex flex-row gap-2 align-items-center">
              <span
                :class="{ 'hover': col.optionLabel === 'name' }"
                @click="col.optionLabel === 'name' ? $emit('updateTransaction', data.id) : null"
              >
                {{ col.optionLabel === "name" ? data[col.field]?.["name"] : data[col.field]?.["ticker"] }}
              </span>
            </div>
          </template>
          <template v-else-if="col.field === 'transaction_type'">
            <div class="flex flex-row gap-2 align-items-center">
              <span :style="{ color: data[col.field] === 'buy' ? colors.pos : colors.neg }">
                {{ data[col.field].charAt(0).toUpperCase() + data[col.field].slice(1) }}
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
