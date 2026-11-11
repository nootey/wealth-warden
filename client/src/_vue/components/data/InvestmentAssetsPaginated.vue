<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import ActionRow from "../layout/ActionRow.vue";
import type { Column } from "../../../services/filter_registry.ts";
import { computed, onMounted, onUnmounted, provide, ref, watch } from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import { useCursorPager } from "../../../utils/use_cursor_pager.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useWsStore } from "../../../services/stores/ws_store.ts";
import type { InvestmentAsset } from "../../../models/investment_models.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";

const props = defineProps<{
  accID?: number;
}>();

defineEmits<{
  updateAsset: [id: number];
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const wsStore = useWsStore();

const loading = ref(false);
const records = ref<InvestmentAsset[]>([]);

const apiPrefix = "investments";
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
const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const pager = useCursorPager(default_rows.value!);
const paginator = pager.paginator;
const sort = ref(filterHelper.initSort());

const activeColumns = computed<Column[]>(() => [
  { field: "ticker", header: "Ticker" },
  { field: "quantity", header: "Quantity" },
  { field: "price", header: "Price", hideOnMobile: true, sortable: false },
  { field: "value_at_buy", header: "Invested", hideOnMobile: true },
  { field: "current_value", header: "Current value", hideOnMobile: true },
  { field: "average_buy_price", header: "Average", hideOnMobile: true },
  { field: "profit_loss", header: "PNL" },
]);

const unsubscribers: (() => void)[] = [];

onMounted(async () => {
  await getData();

  unsubscribers.push(wsStore.on("asset.pnl_synced", () => refresh()));
});

onUnmounted(() => unsubscribers.forEach((unsubscribe) => unsubscribe()));

watch(includeDeleted, async () => {
  pager.reset();
  await getData();
});

async function getData() {
  loading.value = true;

  try {
    const paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      pager.cursor.value,
    );

    records.value = paginationResponse.data;
    pager.update(paginationResponse, records.value.length);
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function onPage(event: any) {
  if (pager.goTo(event.page, event.rows)) await getData();
}

async function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  pager.reset();
  await getData();
}

function refresh() {
  getData();
}

provide("switchSort", switchSort);

defineExpose({ refresh });
</script>

<template>
  <div class="flex flex-col w-full gap-2">
    <ActionRow>
      <template #includeDeleted>
        <div
          v-tooltip="includeDeleted ? 'Hide closed' : 'Show closed'"
          class="hover-icon flex flex-row items-center justify-center p-2 rounded-lg"
          :style="{
            border: includeDeleted
              ? '1px solid var(--accent-primary)'
              : '1px solid var(--border-color)',
          }"
          @click="includeDeleted = !includeDeleted"
        >
          <i
            class="pi pi-eye"
            :style="{
              fontSize: '0.845rem',
              color: includeDeleted ? 'var(--accent-primary)' : undefined,
            }"
          />
        </div>
      </template>
    </ActionRow>
    <DataTable
      data-key="id"
      class="w-full enhanced-table"
      :loading="loading"
      :value="records"
      scrollable
      scroll-height="50vh"
      :row-class="vueHelper.closedRowClass"
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
            :sortable="!!sort && col.sortable !== false"
            @click="!sort"
          />
        </template>
        <template #body="{ data }">
          <template v-if="col.field === 'average_buy_price'">
            <div class="flex flex-row gap-2 items-center">
              <span>{{
                vueHelper.displayAssetPrice(
                  data.average_buy_price,
                  data.investment_type,
                  data.currency,
                  true,
                )
              }}</span>
            </div>
          </template>
          <template v-else-if="col.field === 'price'">
            <div class="flex flex-row gap-2 items-center">
              <span>{{
                vueHelper.displayAssetPrice(
                  data.latest_price,
                  data.investment_type,
                  data.latest_price_currency,
                )
              }}</span>
            </div>
          </template>
          <template
            v-else-if="['value_at_buy', 'current_value'].includes(col.field)"
          >
            <div class="flex flex-row gap-2 items-center">
              <span>{{
                vueHelper.displayAsCurrency(data[col.field], data["currency"])
              }}</span>
            </div>
          </template>
          <template v-else-if="col.field == 'profit_loss'">
            <span
              class="font-medium tabular-nums whitespace-nowrap"
              :style="{
                color:
                  data[col.field] > 0
                    ? colors.pos
                    : data[col.field] < 0
                      ? colors.neg
                      : undefined,
              }"
            >
              {{ data[col.field] > 0 ? "+" : ""
              }}{{
                vueHelper.displayAsCurrency(data[col.field], data["currency"])
              }}
            </span>
          </template>
          <template v-else-if="col.field === 'account'">
            <div class="flex flex-row gap-2 items-center account-row">
              <span
                v-tooltip.top="data[col.field]?.['name']"
                class="truncate-text"
              >
                {{ data[col.field]?.["name"] }}
              </span>
            </div>
          </template>
          <template v-else-if="col.field === 'ticker'">
            <div class="flex flex-row gap-2 items-center account-row">
              <span class="hover" @click="$emit('updateAsset', data.id)">
                {{ data[col.field] }}
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
