<script setup lang="ts">
import { computed, onMounted, provide, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useInvestmentStore } from "../../../services/stores/investment_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import type {
  InvestmentIncome,
  InvestmentType,
} from "../../../models/investment_models.ts";
import type { PaginatorState } from "../../../models/shared_models.ts";
import type { Column } from "../../../services/filter_registry.ts";

const props = defineProps<{
  assetId: number;
  assetCurrency: string;
  investmentType: InvestmentType;
}>();

const toastStore = useToastStore();
const sharedStore = useSharedStore();
const investmentStore = useInvestmentStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const records = ref<InvestmentIncome[]>([]);
const loading = ref(false);
const page = ref(1);
const rows = ref([10, 25, 50]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: rows.value[0]!,
});
const sort = ref(filterHelper.initSort("txn_date"));

const apiPrefix = computed(() => `investments/assets/${props.assetId}/income`);

const columns = computed<Column[]>(() => [
  { field: "income_type", header: "Type" },
  { field: "txn_date", header: "Date", type: "date" },
  ...(props.investmentType === "crypto"
    ? [{ field: "quantity", header: "Quantity", type: "number" } as Column]
    : []),
  { field: "amount", header: "Amount", type: "number" },
  ...(props.investmentType === "stock" || props.investmentType === "etf"
    ? [{ field: "tax_withheld", header: "Tax", type: "number" } as Column]
    : []),
  { field: "notes", header: "Notes", hideOnMobile: true },
]);

async function load(): Promise<void> {
  loading.value = true;
  try {
    const res = await sharedStore.getRecordsPaginated(
      apiPrefix.value,
      { rowsPerPage: paginator.value.rowsPerPage, sort: sort.value },
      page.value,
    );
    records.value = res.data;
    paginator.value.total = res.total_records;
    paginator.value.from = res.from;
    paginator.value.to = res.to;
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function onPage(event: any): Promise<void> {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await load();
}

async function switchSort(column: string): Promise<void> {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  await load();
}

function deleteConfirmation(id: number): void {
  confirm.require({
    header: "Delete record?",
    message: "This will permanently delete this income record.",
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id),
  });
}

async function deleteRecord(id: number): Promise<void> {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }
  try {
    const response = await investmentStore.deleteIncome(id);
    toastStore.successResponseToast(response);
    await load();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function formatType(type: string): string {
  return type === "staking_reward" ? "Staking" : "Dividend";
}

onMounted(load);

provide("switchSort", switchSort);

defineExpose({ refresh: load });
</script>

<template>
  <div
    class="flex flex-column w-full border-round-2xl"
    style="
      padding: 0.25rem 0.25rem 0 0.25rem;
      border: 1px solid var(--border-color);
    "
  >
    <DataTable
      data-key="id"
      class="w-full enhanced-table"
      :loading="loading"
      :value="records"
      scrollable
      scroll-height="30vh"
      column-resize-mode="fit"
      scroll-direction="both"
    >
      <template #empty>
        <div style="padding: 10px">No income events recorded yet.</div>
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
          <template v-if="col.field === 'income_type'">
            {{ formatType(data.income_type) }}
          </template>
          <template v-else-if="col.field === 'txn_date'">
            {{ dateHelper.formatDate(data.txn_date, false) }}
          </template>
          <template v-else-if="col.field === 'amount'">
            {{ vueHelper.displayAsCurrency(data.amount, data.currency) }}
          </template>
          <template v-else-if="col.field === 'tax_withheld'">
            {{ vueHelper.displayAsCurrency(data.tax_withheld, data.currency) }}
          </template>
          <template v-else-if="col.field === 'notes'">
            <span v-tooltip.top="data[col.field]" class="truncate-text">
              {{ data[col.field] ?? "" }}
            </span>
          </template>
          <template v-else>
            {{ data[col.field] ?? "" }}
          </template>
        </template>
      </Column>

      <Column v-if="hasPermission('manage_data')" header="">
        <template #body="{ data }">
          <i
            class="pi pi-trash hover-icon text-sm"
            style="color: var(--p-red-300)"
            @click="deleteConfirmation(data.id)"
          />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped></style>
