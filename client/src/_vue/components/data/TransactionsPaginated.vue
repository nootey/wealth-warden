<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type {Transaction} from "../../../models/transaction_models.ts";
import type {Column} from "../../../services/filter_registry.ts";
import {computed, onMounted, ref, watch} from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import type { SortObj } from "../../../models/shared_models.ts";
import {useChartColors} from "../../../style/theme/chartColors.ts";

const props = defineProps<{
    columns: Column[];
    sort?: SortObj;
    rows?: number[];
    filters?: any;
    includeDeleted?: boolean;
    fetchPage: (args: {
        page: number;
        rows: number;
        sort?: SortObj;
        filters?: any;
        includeDeleted?: boolean;
    }) => Promise<{ data: Transaction[]; total: number }>;
    readOnly: boolean;
}>();

const emits = defineEmits<{
    (e: "onPage", payload: { page: number; rows: number }): void;
    (e: "sortChange", column: string): void;
    (e: "rowClick", id: number): void;
}>();

const { colors } = useChartColors();

const rowsOptions = computed(() => props.rows ?? [25, 50, 100]);
const pageLocal = ref(1);
const rowsPerPage = ref(rowsOptions.value[0]);
const total = ref(0);
const localSort = ref<SortObj>(props.sort ?? filterHelper.initSort());

const recordsLocal = ref<Transaction[]>([]);
const loading = ref(false);
const requestSeq = ref(0);

const derivedPaginator = computed(() => {
    const from = total.value === 0 ? 0 : (pageLocal.value - 1) * rowsPerPage.value + 1;
    const to = Math.min(pageLocal.value * rowsPerPage.value, total.value);
    return { total: total.value, from, to, rowsPerPage: rowsPerPage.value };
});

async function getData() {
    loading.value = true;
    const mySeq = ++requestSeq.value;

    try {
        const res = await props.fetchPage({
            page: pageLocal.value,
            rows: rowsPerPage.value,
            sort: localSort.value,
            filters: props.filters,
            includeDeleted: props.includeDeleted,
        });

        // Ignore stale responses
        if (mySeq !== requestSeq.value) return;

        recordsLocal.value = res.data;
        total.value = res.total ?? 0;
    } finally {
        if (mySeq === requestSeq.value) loading.value = false;
    }
}

onMounted(getData);

watch(
    () => [props.sort?.field, props.sort?.order, props.filters, props.includeDeleted],
    () => { pageLocal.value = 1; getData(); }
);

function handlePage(e: { page: number; rows: number }) {
    pageLocal.value = e.page + 1;
    rowsPerPage.value = e.rows;
    emits("onPage", { page: pageLocal.value, rows: rowsPerPage.value });
    getData();
}

function triggerSort(col: string) {
    emits("sortChange", col);
}

function refresh() { getData(); }

defineExpose({ refresh });

</script>

<template>
  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :loading="loading"
    :value="recordsLocal"
    scrollable
    scroll-height="50vh"
    :row-class="vueHelper.deletedRowClass"
    column-resize-mode="fit"
    scroll-direction="both"
  >
    <template #empty>
      <div style="padding: 10px;">
        No records found.
      </div>
    </template>
    <template #loading>
      <LoadingSpinner />
    </template>
    <template #footer>
      <CustomPaginator
        :paginator="derivedPaginator"
        :rows="rowsOptions"
        @on-page="handlePage"
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
          :sort="localSort"
          :sortable="!!sort"
          @click="!sort && triggerSort(col.field as string)"
        />
      </template>
      <template #body="{ data }">
        <template v-if="col.field === 'amount'">
          <div class="flex flex-row gap-2 align-items-center">
            <i
              class="text-xs"
              :class="((data.transaction_type === 'expense' ? data.amount * -1 : data.amount) >= 0)
                ? 'pi pi-angle-up': 'pi pi-angle-down'"
              :style="{ color: ((data.transaction_type === 'expense' ? data.amount * -1 : data.amount) >= 0)
                ? colors.pos: colors.neg }"
            />
            <span>{{ vueHelper.displayAsCurrency(data.transaction_type == "expense" ? (data.amount*-1) : data.amount) }}</span>
          </div>
        </template>
        <template v-else-if="col.field === 'txn_date'">
          {{ dateHelper.combineDateAndTime(data?.txn_date, data?.created_at) }}
        </template>
        <template v-else-if="col.field === 'account'">
          <div class="flex flex-row gap-2 align-items-center account-row">
            <span
              class="hover"
              @click="$emit('rowClick', data.id)"
            >
              {{ data[col.field]["name"] }}
            </span>
            <i
              v-if="data[col.field]['deleted_at']"
              v-tooltip="'This account is closed.'"
              class="pi pi-ban popup-icon hover-icon"
            />
          </div>
        </template>
        <template v-else-if="col.field === 'category'">
          {{ data[col.field]["display_name"] }}
        </template>
        <template v-else-if="col.field === 'description'">
          <span
            v-tooltip.top="data[col.field]"
            class="truncate-text"
          >
            {{ data[col.field] }}
          </span>
        </template>
        <template v-else>
          {{ data[col.field] }}
        </template>
      </template>
    </Column>
  </DataTable>
</template>

<style scoped>
.hover { font-weight: bold; }
.hover:hover { cursor: pointer; text-decoration: underline; }

.account-row .popup-icon {
    opacity: 0;
    transition: opacity .15s ease;
}
.account-row:hover .popup-icon {
    opacity: 1;
}

.account-row.advanced .popup-icon {
    opacity: 1;
}

</style>