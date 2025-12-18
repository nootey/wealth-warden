<script setup lang="ts">
import TransactionForm from "../components/forms/TransactionForm.vue";
import {computed, onMounted, provide, ref} from "vue";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import ActionRow from "../components/layout/ActionRow.vue";
import type {FilterObj} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import filterHelper from "../../utils/filter_helper.ts";
import type {Category} from "../../models/transaction_models.ts";
import type {Column} from "../../services/filter_registry.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import type {Account} from "../../models/account_models.ts";
import TransfersPaginated from "../components/data/TransfersPaginated.vue";
import TransactionsPaginated from "../components/data/TransactionsPaginated.vue";
import {useRouter} from "vue-router";
import {usePermissions} from "../../utils/use_permissions.ts";
import TransactionTemplates from "../features/TransactionTemplates.vue";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

onMounted(async () => {
    await transactionStore.getCategories();
    await accountStore.getAllAccounts(false, true);
    await getTrTemplateCount();
})

const router = useRouter();
const { hasPermission } = usePermissions();

const apiPrefix = "transactions";

const trRef = ref<InstanceType<typeof TransfersPaginated> | null>(null);
const txRef = ref<InstanceType<typeof TransactionsPaginated> | null>(null);
const tpRef = ref<InstanceType<typeof TransactionTemplates> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const templateModal = ref(false);
const updateTransactionID = ref(null);
const includeDeleted = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);
const trTemplateCount = ref<number>(0);

const sort = ref(filterHelper.initSort("txn_date"));
const filterStorageIndex = ref(apiPrefix+"-filters");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"));
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
  { field: 'account', header: 'Account', type: 'enum', options: accounts.value, optionLabel: 'name'},
  { field: 'category', header: 'Category', type: 'enum', options: categories.value, optionLabel: 'name', hideOnMobile: true },
  { field: 'amount', header: 'Amount', type: "number" },
  { field: 'txn_date', header: 'Date', type: "date" },
  { field: 'description', header: 'Description', type: "text", hideOnMobile: true },
]);

async function getTrTemplateCount() {

    try {
        let res = await transactionStore.getTransactionTemplateCount();
        trTemplateCount.value = res.data;
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

async function loadTransactionsPage({ page, rows, sort: s, filters: f, include_deleted }: any) {
    let response = null;

    try {
        response =  await sharedStore.getRecordsPaginated(
            apiPrefix,
            { rowsPerPage: rows, sort: s, filters: f, include_deleted },
            page
        );
    } catch (e) {
        toastStore.errorResponseToast(e);
    }

    return { data: response?.data, total: response?.total_records };
}

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case 'addTransaction': {
        if(!hasPermission("manage_data")) {
            toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
            return;
        }
        createModal.value = value;
        break;
    }
    case 'openTemplateView': {
        if(!hasPermission("manage_data")) {
            toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
            return;
        }
        templateModal.value = value;
        break;
    }
    case 'updateTransaction': {
        if(!hasPermission("manage_data")) {
            toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
            return;
        }
        updateModal.value = true;
        updateTransactionID.value = value;
        break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'completeTxOperation': {
        createModal.value = false;
        updateModal.value = false;
        txRef.value?.refresh();
        break;
    }
    case 'completeTrOperation': {
        createModal.value = false;
        trRef.value?.refresh();
        break;
    }
    case 'refreshTemplateCount': {
        await getTrTemplateCount();
        break;
    }
    case 'deleteTxn': {
        createModal.value = false;
        updateModal.value = false;
        txRef.value?.refresh();
        break;
    }
    default: {
      break;
    }
  }
}

function applyFilters(list: FilterObj[]){
  filters.value = filterHelper.mergeFilters(filters.value, list);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  txRef.value?.refresh();
  filterOverlayRef.value.hide();
}

function clearFilters(){
  filters.value = [];
  localStorage.removeItem(filterStorageIndex.value);
  cancelFilters();
  txRef.value?.refresh();
}

function cancelFilters(){
  filterOverlayRef.value.hide();
}

function removeFilter(index: number) {
  if (index < 0 || index >= filters.value.length) return;

  const next = filters.value.slice();
  next.splice(index, 1);
  filters.value = next;

  if (filters.value.length > 0) {
    localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  } else {
    localStorage.removeItem(filterStorageIndex.value);
  }

  txRef.value?.refresh();
}

function switchSort(column:string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
}

function toggleFilterOverlay(event: any) {
  filterOverlayRef.value.toggle(event);
}

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);

</script>

<template>
  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{'501px': '90vw'}"
    :modal="true"
    :style="{width: '500px'}"
    header="Add transaction"
  >
    <TransactionForm
      mode="create"
      @complete-tx-operation="handleEmit('completeTxOperation')"
      @complete-tr-operation="handleEmit('completeTrOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{'501px': '90vw'}"
    :modal="true"
    :style="{width: '500px'}"
    header="Transaction details"
  >
    <TransactionForm
      mode="update"
      :record-id="updateTransactionID"
      @complete-tx-operation="handleEmit('completeTxOperation')"
      @complete-tx-delete="handleEmit('deleteTxn')"
    />
  </Dialog>

  <Dialog
    v-model:visible="templateModal"
    class="rounded-dialog"
    :breakpoints="{'901px': '90vw'}"
    :modal="true"
    :style="{width: '900px'}"
    header="Transaction templates"
  >
    <TransactionTemplates
      ref="tpRef"
      @refresh-template-count="handleEmit('refreshTemplateCount')"
    />
  </Dialog>

  <Popover
    ref="filterOverlayRef"
    class="rounded-popover"
    :style="{width: '420px'}"
    :breakpoints="{'775px': '90vw'}"
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

  <main
    class="flex flex-column w-full p-2 align-items-center"
    style="height: 100%;"
  >
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="border: 1px solid var(--border-color); background: var(--background-secondary); max-width: 1000px;"
    >
      <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
        <div style="font-weight: bold;">
          Transactions
        </div>
        <i
          v-if="hasPermission('manage_data')"
          v-tooltip="'Go to categories settings.'"
          class="pi pi-external-link hover-icon mr-auto text-sm"
          @click="router.push('settings/categories')"
        />
        <Button
          class="outline-button"
          @click="manipulateDialog('openTemplateView', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-database" />
            <span><span class="mobile-hide"> Templates </span> {{ "(" + trTemplateCount + ")" }}</span>
          </div>
        </Button>
        <Button
          class="main-button"
          @click="manipulateDialog('addTransaction', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Transaction </span>
          </div>
        </Button>
      </div>

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
              style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)"
              @click="toggleFilterOverlay($event)"
            >
              <i
                class="pi pi-filter"
                style="font-size: 0.845rem"
              />
              <div>Filter</div>
            </div>
          </template>
          <template #includeDeleted>
            <div
              class="flex align-items-center gap-2"
              style="margin-left: auto;"
            >
              <span style="font-size: 0.8rem;">Include deleted</span>
              <ToggleSwitch v-model="includeDeleted" />
            </div>
          </template>
        </ActionRow>
      </div>

      <div
        id="mobile-row"
        class="flex flex-row w-full"
      >
        <TransactionsPaginated
          ref="txRef"
          :read-only="false"
          :columns="activeColumns"
          :sort="sort"
          :filters="filters"
          :include-deleted="includeDeleted"
          :fetch-page="loadTransactionsPage"
          @sort-change="switchSort"
          @row-click="(id) => manipulateDialog('updateTransaction', id)"
        />
      </div>

      <label>Transfers</label>
      <div
        id="mobile-row"
        class="flex flex-row w-full"
      >
        <TransfersPaginated ref="trRef" />
      </div>
    </div>
  </main>
</template>

<style scoped>

</style>