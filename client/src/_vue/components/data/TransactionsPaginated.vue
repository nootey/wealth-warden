<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type {
  Transaction,
  TransactionBatchTotals,
} from "../../../models/transaction_models.ts";
import type { Column } from "../../../services/filter_registry.ts";
import { computed, onMounted, provide, ref, watch } from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import type {
  FilterObj,
  PaginatorState,
} from "../../../models/shared_models.ts";
import FilterMenu from "../filters/FilterMenu.vue";
import FilterPopover from "../filters/FilterPopover.vue";
import ActiveFilters from "../filters/ActiveFilters.vue";
import ActionRow from "../layout/ActionRow.vue";
import type { UserSettings } from "../../../models/settings_models.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import { isBulkSelectable } from "../../../models/transaction_models.ts";

const props = defineProps<{
  columns: Column[];
  readOnly: boolean;
  accID?: number;
}>();

defineEmits<{
  rowClick: [id: number];
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const settingsStore = useSettingsStore();
const transactionStore = useTransactionStore();
const { hasPermission } = usePermissions();
const confirm = useConfirm();
const { colors } = useChartColors();

const loading = ref(false);
const records = ref<Transaction[]>([]);
const totals = ref<TransactionBatchTotals | null>(null);

const canManage = computed(() => hasPermission("manage_data"));
const selecting = ref(false);
const selected = ref<Transaction[]>([]);
const bulkLoading = ref(false);
const categoryDialog = ref(false);
const descriptionDialog = ref(false);
const bulkCategoryId = ref<number | null>(null);
const bulkDescription = ref("");

const categoryOptions = computed(() =>
  transactionStore.categories
    .filter((c) => c.parent_id != null || c.name === "(uncategorized)")
    .map((c) => ({
      label: c.display_name || c.name,
      value: c.id,
      classification: c.classification,
    })),
);

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
const rows = ref([10, 25, 50, 100]);
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
const userSettings = ref<UserSettings>();

onMounted(async () => {
  await getSettings();
  await getData();
});

watch(includeDeleted, async () => {
  await getData(1); // Reset to page 1 when toggling
});

async function getSettings() {
  try {
    const res = await settingsStore.getUserSettings();
    userSettings.value = res.data;
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function getData(new_page: number | null = null) {
  loading.value = true;
  if (new_page) page.value = new_page;

  try {
    const paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
      page.value,
    );
    records.value = paginationResponse.data.records;
    totals.value = paginationResponse.data.totals;
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
  if (props.readOnly) return;

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
  if (props.readOnly) return;
  filterOverlayRef.value.toggle(event);
}

function refresh() {
  getData();
}

function toggleSelecting() {
  selecting.value = !selecting.value;
  if (!selecting.value) selected.value = [];
}

function isDataSelectable(event: { data: Transaction }) {
  return isBulkSelectable(event.data.transaction_type);
}

async function openCategoryDialog() {
  bulkCategoryId.value = null;
  if (transactionStore.categories.length === 0) {
    try {
      await transactionStore.getCategories();
    } catch (e) {
      toastStore.errorResponseToast(e);
    }
  }
  categoryDialog.value = true;
}

function openDescriptionDialog() {
  bulkDescription.value = "";
  descriptionDialog.value = true;
}

function confirmBulkDelete() {
  if (selected.value.length === 0) return;
  confirm.require({
    header: "Delete transactions?",
    message: `This will delete ${selected.value.length} selected transaction(s).`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => runBulk("delete", {}),
  });
}

async function runBulk(
  action: "set_category" | "set_description" | "delete",
  extra: { category_id?: number; description?: string },
) {
  const ids = selected.value
    .map((t) => t.id)
    .filter((id): id is number => id != null);
  if (ids.length === 0) return;

  bulkLoading.value = true;
  try {
    const res = await transactionStore.bulkOperateTransactions({
      ids,
      action,
      ...extra,
    });
    toastStore.successResponseToast(res);
    selected.value = [];
    selecting.value = false;
    categoryDialog.value = false;
    descriptionDialog.value = false;
    await getData();
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    bulkLoading.value = false;
  }
}

provide("removeFilter", removeFilter);
provide("switchSort", switchSort);

defineExpose({ refresh });
</script>

<template>
  <FilterPopover ref="filterOverlayRef">
    <FilterMenu
      v-model:value="filters"
      :columns="props.columns"
      :api-source="apiPrefix"
      @apply="(list) => applyFilters(list)"
      @clear="clearFilters"
      @cancel="cancelFilters"
    />
  </FilterPopover>

  <div class="flex flex-col w-full gap-4">
    <div
      id="balance-row"
      class="flex w-full p-4 gap-2 rounded-xl bordered justify-between items-center"
      style="max-width: 1000px; border: 1px solid var(--border-color)"
    >
      <div
        class="flex-1 text-center px-4"
        style="border-right: 1px solid var(--border-color)"
      >
        <div class="text-sm" style="color: var(--text-secondary)">Total</div>
        <div class="font-bold">
          {{ totals?.count }}
        </div>
      </div>
      <div
        class="flex-1 text-center px-4"
        style="border-right: 1px solid var(--border-color)"
      >
        <div class="text-sm" style="color: var(--text-secondary)">Income</div>
        <div class="font-bold">
          {{ vueHelper.displayAsCurrency(totals?.income!) }}
        </div>
      </div>
      <div class="flex-1 text-center px-4">
        <div class="text-sm" style="color: var(--text-secondary)">Expenses</div>
        <div class="font-bold">
          {{ vueHelper.displayAsCurrency(totals?.expenses!) }}
        </div>
      </div>
    </div>

    <div v-if="!readOnly" class="flex flex-row w-full">
      <ActionRow pills-last>
        <template #activeFilters>
          <ActiveFilters
            :active-filters="filters"
            :show-only-active="false"
            active-filter=""
          />
        </template>
        <template #filterButton>
          <div
            v-tooltip="'Filter'"
            class="hover-icon flex flex-row items-center justify-center"
            style="
              padding: 0.5rem;
              border-radius: 8px;
              border: 1px solid var(--border-color);
            "
            @click="toggleFilterOverlay($event)"
          >
            <i class="pi pi-filter" style="font-size: 0.845rem" />
          </div>
        </template>
        <template #includeDeleted>
          <div
            v-tooltip="includeDeleted ? 'Hide archived' : 'Show archived'"
            class="hover-icon flex flex-row items-center justify-center"
            :style="{
              padding: '0.5rem',
              borderRadius: '8px',
              border: includeDeleted
                ? '1px solid var(--accent-primary)'
                : '1px solid var(--border-color)',
            }"
            @click="includeDeleted = !includeDeleted"
          >
            <i
              class="pi pi-trash"
              :style="{
                fontSize: '0.845rem',
                color: includeDeleted ? 'var(--accent-primary)' : undefined,
              }"
            />
          </div>
        </template>
        <template v-if="canManage" #selectButton>
          <div
            v-tooltip="selecting ? 'Cancel selection' : 'Select'"
            class="hover-icon flex flex-row items-center justify-center"
            style="
              padding: 0.5rem;
              border-radius: 8px;
              border: 1px solid var(--border-color);
            "
            @click="toggleSelecting"
          >
            <i
              :class="selecting ? 'pi pi-times' : 'pi pi-check-square'"
              style="font-size: 0.845rem"
            />
          </div>
        </template>
      </ActionRow>
    </div>

    <div
      v-if="!readOnly && canManage && selecting"
      class="flex flex-row items-center gap-2"
    >
      <span style="font-size: 0.8rem; color: var(--text-secondary)">
        {{ selected.length }} selected
      </span>
      <Button
        class="outline-button text-sm"
        :disabled="selected.length === 0 || bulkLoading"
        @click="openCategoryDialog"
      >
        <span>Category</span>
      </Button>
      <Button
        class="outline-button text-sm"
        :disabled="selected.length === 0 || bulkLoading"
        @click="openDescriptionDialog"
      >
        <span>Description</span>
      </Button>
      <Button
        class="outline-button text-sm"
        severity="danger"
        :disabled="selected.length === 0 || bulkLoading"
        @click="confirmBulkDelete"
      >
        <span>Delete</span>
      </Button>
    </div>

    <div
      class="flex flex-col w-full rounded-2xl"
      style="
        padding: 0.25rem 0.25rem 0 0.25rem;
        border: 1px solid var(--border-color);
      "
    >
      <DataTable
        v-model:selection="selected"
        data-key="id"
        class="w-full enhanced-table"
        :loading="loading"
        :value="records"
        :is-data-selectable="isDataSelectable"
        scrollable
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
          v-if="selecting"
          selection-mode="multiple"
          header-style="width: 3rem"
        />

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
              <div class="flex flex-row gap-2 items-center">
                <i
                  class="text-xs"
                  :class="
                    (data.direction === 'expense'
                      ? data.amount * -1
                      : data.amount) >= 0
                      ? 'pi pi-angle-up'
                      : 'pi pi-angle-down'
                  "
                  :style="{
                    color:
                      (data.direction === 'expense'
                        ? data.amount * -1
                        : data.amount) >= 0
                        ? colors.pos
                        : colors.neg,
                  }"
                />
                <span>{{
                  vueHelper.displayAsCurrency(
                    data.direction == "expense"
                      ? data.amount * -1
                      : data.amount,
                  )
                }}</span>
              </div>
            </template>
            <template v-else-if="col.field === 'txn_date'">
              {{ dateHelper.formatDate(data.txn_date, false) }}
            </template>
            <template v-else-if="col.field === 'account'">
              <div class="flex flex-row gap-2 items-center account-row">
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
  </div>

  <Dialog
    v-model:visible="categoryDialog"
    class="rounded-dialog"
    :modal="true"
    :breakpoints="{ '501px': '90vw' }"
    :style="{ width: '400px' }"
    header="Set category"
  >
    <div class="flex flex-col gap-4">
      <Select
        v-model="bulkCategoryId"
        class="w-full"
        :options="categoryOptions"
        option-label="label"
        option-value="value"
        filter
        placeholder="Select a category"
      >
        <template #option="{ option }">
          <div class="flex justify-between w-full gap-2">
            <span>{{ option.label }}</span>
            <small class="text-muted-color">{{ option.classification }}</small>
          </div>
        </template>
      </Select>
      <small style="color: var(--text-secondary)">
        Income and expense categories also overwrite each transaction's
        direction to match.
      </small>
      <Button
        class="main-button"
        :disabled="bulkCategoryId == null || bulkLoading"
        :loading="bulkLoading"
        @click="runBulk('set_category', { category_id: bulkCategoryId! })"
      >
        Apply to {{ selected.length }}
      </Button>
    </div>
  </Dialog>

  <Dialog
    v-model:visible="descriptionDialog"
    class="rounded-dialog"
    :modal="true"
    :breakpoints="{ '501px': '90vw' }"
    :style="{ width: '400px' }"
    header="Set description"
  >
    <div class="flex flex-col gap-4">
      <InputText
        v-model="bulkDescription"
        class="w-full"
        placeholder="New description"
      />
      <Button
        class="main-button"
        :disabled="bulkDescription.trim().length === 0 || bulkLoading"
        :loading="bulkLoading"
        @click="runBulk('set_description', { description: bulkDescription })"
      >
        Apply to {{ selected.length }}
      </Button>
    </div>
  </Dialog>
</template>

<style scoped>
@media (max-width: 768px) {
  #balance-row {
    padding: 1rem !important;
    font-size: 80%;
  }
}

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
