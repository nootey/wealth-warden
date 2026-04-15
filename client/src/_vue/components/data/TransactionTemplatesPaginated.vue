<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import { computed, onMounted, provide, ref, watch } from "vue";
import type {
  TemplateSummary,
  TransactionTemplate,
} from "../../../models/transaction_models.ts";
import filterHelper from "../../../utils/filter_helper.ts";
import type { Column } from "../../../services/filter_registry.ts";
import dateHelper from "../../../utils/date_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import TransactionTemplateForm from "../forms/TransactionTemplateForm.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import type { PaginatorState } from "../../../models/shared_models.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";

const emit = defineEmits<{
  (event: "refreshTemplateCount"): void;
}>();

const sharedStore = useSharedStore();
const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const confirm = useConfirm();
const { colors } = useChartColors();

const apiPrefix = "transactions/templates";

const tabOptions = [
  { label: "Transactions", value: "transaction" },
  { label: "Transfers", value: "transfer" },
];
const activeTab = ref<"transaction" | "transfer">("transaction");

onMounted(async () => {
  await Promise.all([getData(), getSummary()]);
});

watch(activeTab, async () => {
  page.value = 1;
  sort.value = filterHelper.initSort();
  await getData();
});

const loadingRecords = ref(true);
const loadingSummary = ref(true);
const records = ref<TransactionTemplate[]>([]);
const summary = ref<TemplateSummary | null>(null);
const createModal = ref(false);
const updateModal = ref(false);
const updateRecordID = ref(null);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: null,
    template_type: activeTab.value,
  };
});
const rows = ref([10, 25]);
const default_rows = ref(rows.value[0]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort());

const activeColumns = computed<Column[]>(() => {
  if (activeTab.value === "transfer") {
    return [
      { field: "name", header: "Name" },
      { field: "account", header: "From" },
      { field: "to_account", header: "To" },
      { field: "amount", header: "Amount" },
      { field: "frequency", header: "Frequency" },
      { field: "next_run_at", header: "Next run" },
    ];
  }
  return [
    { field: "name", header: "Name" },
    { field: "account", header: "Account" },
    { field: "category", header: "Category" },
    { field: "amount", header: "Amount" },
    { field: "frequency", header: "Frequency" },
    { field: "next_run_at", header: "Next run" },
  ];
});

async function getSummary() {
  loadingSummary.value = true;
  try {
    const response = await transactionStore.getTemplateSummary();
    summary.value = response.data;
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loadingSummary.value = false;
  }
}

async function getData(new_page = null) {
  loadingRecords.value = true;
  if (new_page) page.value = new_page;

  try {
    let paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      page.value,
    );
    records.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingRecords.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

async function deleteConfirmation(id: number, name: string) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete template: ${name}".`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id),
  });
}

async function deleteRecord(id: number) {
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
    await Promise.all([getData(), getSummary()]);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function refresh() {
  getData();
}

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "addTemplate": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      createModal.value = value;
      break;
    }
    case "updateTemplate": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      updateModal.value = true;
      updateRecordID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any, data?: any) {
  switch (emitType) {
    case "completeOperation": {
      createModal.value = false;
      updateModal.value = false;
      await Promise.all([getData(), getSummary()]);
      emit("refreshTemplateCount");
      break;
    }
    case "updateTemplate": {
      updateModal.value = true;
      updateRecordID.value = data;
      break;
    }
    default: {
      break;
    }
  }
}

async function toggleActiveTemplate(
  tp: TransactionTemplate,
  nextValue: boolean,
): Promise<boolean> {
  const previous = tp.is_active;

  try {
    tp.is_active = nextValue;

    const response = await transactionStore.toggleTemplateActiveState(tp.id!);
    toastStore.successResponseToast(response);

    emit("refreshTemplateCount");
    await Promise.all([getData(), getSummary()]);
    return true;
  } catch (error) {
    // add a small delay for the toggle animation to complete
    await new Promise((resolve) => setTimeout(resolve, 300));
    tp.is_active = previous;
    toastStore.errorResponseToast(error);
    return false;
  }
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

provide("switchSort", switchSort);

defineExpose({ refresh });
</script>

<template>
  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add template"
  >
    <TransactionTemplateForm
      mode="create"
      :template-type="activeTab"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Template details"
  >
    <TransactionTemplateForm
      mode="update"
      :record-id="updateRecordID"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <div
    class="flex flex-column justify-content-center w-full gap-3"
    style="max-width: 1000px"
  >
    <div
      class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full"
    >
      <SelectButton
        v-model="activeTab"
        :options="tabOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        size="small"
      />
      <Button
        class="main-button ml-auto"
        @click="manipulateDialog('addTemplate', true)"
      >
        <div class="flex flex-row gap-1 align-items-center">
          <i class="pi pi-plus" />
          <span> New </span>
          <span class="mobile-hide"> Template </span>
        </div>
      </Button>
    </div>

    <div
      id="projection-bar"
      class="flex w-full p-3 gap-2 border-round-xl justify-content-between align-items-center"
      style="border: 1px solid var(--border-color)"
    >
      <div
        class="flex-1 text-center px-3"
        style="border-right: 1px solid var(--border-color)"
      >
        <div
          id="projection-label"
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ "Projected income" }}
        </div>
        <div class="font-bold" :style="{ color: colors.pos }">
          {{
            loadingSummary
              ? "—"
              : vueHelper.displayAsCurrency(
                  Number(summary?.monthly_income ?? 0) +
                    Number(summary?.this_month_income ?? 0) || 0,
                )
          }}
        </div>
        <div
          v-if="!loadingSummary && Number(summary?.this_month_income ?? 0) > 0"
          class="text-xs mt-1"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.displayAsCurrency(summary?.monthly_income ?? 0) }} avg.
        </div>
      </div>
      <div
        class="flex-1 text-center px-3"
        style="border-right: 1px solid var(--border-color)"
      >
        <div
          id="projection-label"
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ "Projected expenses" }}
        </div>
        <div class="font-bold" :style="{ color: colors.neg }">
          {{
            loadingSummary
              ? "—"
              : vueHelper.displayAsCurrency(
                  Number(summary?.monthly_expense ?? 0) +
                    Number(summary?.this_month_expense ?? 0) || 0,
                )
          }}
        </div>
        <div
          v-if="!loadingSummary && Number(summary?.this_month_expense ?? 0) > 0"
          class="text-xs mt-1"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.displayAsCurrency(summary?.monthly_expense ?? 0) }} avg.
        </div>
      </div>
      <div class="flex-1 text-center px-3">
        <div
          id="projection-label"
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ "Projected transfers" }}
        </div>
        <div class="font-bold">
          {{
            loadingSummary
              ? "—"
              : vueHelper.displayAsCurrency(
                  Number(summary?.monthly_transfer ?? 0) +
                    Number(summary?.this_month_transfer ?? 0) || 0,
                )
          }}
        </div>
        <div
          v-if="
            !loadingSummary && Number(summary?.this_month_transfer ?? 0) > 0
          "
          class="text-xs mt-1"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.displayAsCurrency(summary?.monthly_transfer ?? 0) }} avg.
        </div>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <DataTable
        class="w-full enhanced-table"
        size="small"
        data-key="id"
        :loading="loadingRecords"
        :value="records"
        scrollable
        :row-class="vueHelper.isActiveRowClass"
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
          style="width: 25%"
        >
          <template #header>
            <ColumnHeader
              :header="col.header"
              :field="col.field"
              :sortable="true"
              :sort="sort"
            />
          </template>
          <template #body="{ data }">
            <template
              v-if="col.field === 'next_run_at' || col.field === 'end_date'"
            >
              {{ dateHelper.formatDate(data[col.field], false) }}
            </template>
            <template v-else-if="col.field === 'name'">
              <span
                class="hover"
                @click="handleEmit('updateTemplate', data.id)"
              >
                {{ data[col.field] }}
              </span>
            </template>
            <template
              v-else-if="col.field === 'account' || col.field === 'to_account'"
            >
              {{ data[col.field]?.name }}
            </template>
            <template v-else-if="col.field === 'amount'">
              <div class="flex flex-row gap-2 align-items-center">
                <i
                  class="text-xs"
                  :class="
                    (data.transaction_type === 'expense'
                      ? data.amount * -1
                      : data.amount) >= 0
                      ? 'pi pi-angle-down'
                      : 'pi pi-angle-up'
                  "
                  :style="{
                    color:
                      (data.transaction_type === 'expense'
                        ? data.amount * -1
                        : data.amount) >= 0
                        ? colors.neg
                        : colors.pos,
                  }"
                />
                <span>
                  {{
                    vueHelper.displayAsCurrency(
                      data.transaction_type == "expense"
                        ? data.amount * -1
                        : data.amount,
                    )
                  }}
                </span>
              </div>
            </template>
            <template v-else-if="col.field === 'category'">
              {{ data[col.field]?.display_name }}
            </template>
            <template
              v-else-if="
                col.field === 'transaction_type' || col.field === 'frequency'
              "
            >
              {{ vueHelper.capitalize(data[col.field]) }}
            </template>
            <template v-else>
              {{ data[col.field] }}
            </template>
          </template>
        </Column>

        <Column header="Actions">
          <template #body="{ data }">
            <div class="flex flex-row align-items-center gap-2">
              <ToggleSwitch
                v-if="hasPermission('manage_data')"
                style="transform: scale(0.675)"
                :model-value="data.is_active"
                @update:model-value="(v) => toggleActiveTemplate(data, v)"
              />
              <i
                v-if="hasPermission('manage_data')"
                class="pi pi-trash hover-icon"
                style="font-size: 0.875rem; color: var(--p-red-300)"
                @click="deleteConfirmation(data?.id, data?.name)"
              />
              <i
                v-else
                v-tooltip="'No action available'"
                class="pi pi-exclamation-circle"
                style="font-size: 0.875rem"
              />
            </div>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #projection-bar {
    padding: 0.5rem !important;
    font-size: 75%;
  }
  #projection-label {
    font-size: 0.75rem !important;
  }
}

.hover {
  font-weight: bold;
}
.hover:hover {
  cursor: pointer;
  text-decoration: underline;
}
</style>
