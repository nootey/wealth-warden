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
import {useConfirm} from "primevue/useconfirm";
import {useAccountStore} from "../../services/stores/account_store.ts";
import type {Account} from "../../models/account_models.ts";
import TransfersPaginated from "../components/data/TransfersPaginated.vue";
import TransactionsPaginated from "../components/data/TransactionsPaginated.vue";
import {useRouter} from "vue-router";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

onMounted(async () => {
    await transactionStore.getCategories();
    await accountStore.getAllAccounts();
})

const confirm = useConfirm();
const router = useRouter();
const apiPrefix = "transactions";

const trRef = ref<InstanceType<typeof TransfersPaginated> | null>(null);
const txRef = ref<InstanceType<typeof TransactionsPaginated> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const updateTransactionID = ref(null);
const includeDeleted = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);

const sort = ref(filterHelper.initSort());
const filterStorageIndex = ref(apiPrefix+"-filters");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"));
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
  { field: 'account', header: 'Account', type: 'enum', options: accounts.value, optionLabel: 'name'},
  { field: 'category', header: 'Category', type: 'enum', options: categories.value, optionLabel: 'name'},
  { field: 'amount', header: 'Amount', type: "number" },
  { field: 'txn_date', header: 'Date', type: "date" },
  { field: 'description', header: 'Description', type: "text" },
]);

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
      createModal.value = value;
      break;
    }
    case 'updateTransaction': {
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

async function deleteConfirmation(id: number, tx_type: string) {
    const txt = tx_type === "transfer" ? tx_type : "transaction";
    confirm.require({
        header: 'Delete record?',
        message: `This will delete transaction: "${txt} : ${id}".`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Delete', severity: 'danger' },
        accept: () => deleteRecord(id, tx_type),
    });
}

async function deleteRecord(id: number, tx_type: string) {
  try {
    let response = await sharedStore.deleteRecord(
        tx_type === "transfer" ? "transactions/transfers" : "transactions",
        id,
    );
    toastStore.successResponseToast(response);
    txRef.value?.refresh();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);

</script>

<template>

  <Dialog class="rounded-dialog" v-model:visible="createModal" :breakpoints="{'501px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Add transaction">
    <TransactionForm mode="create"
                     @completeTxOperation="handleEmit('completeTxOperation')"
                     @completeTrOperation="handleEmit('completeTrOperation')"></TransactionForm>
  </Dialog>

  <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal" :breakpoints="{'501px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Transaction details">
    <TransactionForm mode="update" :recordId="updateTransactionID"
                     @completeTxOperation="handleEmit('completeTxOperation')"></TransactionForm>
  </Dialog>

  <Popover ref="filterOverlayRef" class="rounded-popover">
    <div class="flex flex-column gap-2" style="width: 400px">
      <FilterMenu
          v-model:value="filters"
          :columns="activeColumns"
          :apiSource="apiPrefix"
          @apply="(list) => applyFilters(list)"
          @clear="clearFilters"
          @cancel="cancelFilters"
      />
    </div>
  </Popover>

  <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

    <div class="flex flex-row justify-content-between align-items-center p-3  gap-2 w-full"
         style="border-top-right-radius: 8px; border-top-left-radius: 8px;
         border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px;">

      <div style="font-weight: bold;">Transactions</div>
      <i class="pi pi-map hover-icon mr-auto text-sm" @click="router.push('settings/categories')" v-tooltip="'Go to categories settings.'"></i>

      <Button label="New transaction" icon="pi pi-plus" class="main-button" @click="manipulateDialog('addTransaction', true)"></Button>
    </div>

    <div class="flex flex-column justify-content-center p-3 w-full gap-3"
         style="border-bottom-right-radius: 8px; border-bottom-left-radius: 8px;
         border: 1px solid var(--border-color); background: var(--background-secondary);
         max-width: 1000px;">

      <div class="flex flex-row w-full">
        <ActionRow>
            <template #activeFilters>
            <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />
            </template>
              <template #filterButton>
                <div class="hover-icon flex flex-row align-items-center gap-2" @click="toggleFilterOverlay($event)"
                     style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)">
                  <i class="pi pi-filter" style="font-size: 0.845rem"></i>
                  <div>Filter</div>
                </div>
              </template>
            <template #includeDeleted>
                <div class="flex align-items-center gap-2" style="margin-left: auto;">
                    <span style="font-size: 0.8rem;">Include deleted</span>
                    <ToggleSwitch v-model="includeDeleted" />
                </div>
            </template>
        </ActionRow>
      </div>

        <div class="flex flex-row gap-2 w-full">
            <TransactionsPaginated
                    ref="txRef"
                    :readOnly="false"
                    :columns="activeColumns"
                    :sort="sort"
                    :filters="filters"
                    :include_deleted="includeDeleted"
                    :fetchPage="loadTransactionsPage"
                    @sortChange="switchSort"
                    @rowClick="(id) => manipulateDialog('updateTransaction', id)"
                    @deleteClick="({ id, tx_type }) => deleteConfirmation(id, tx_type)"
            />
        </div>

        <label>Transfers</label>
        <div class="flex flex-row gap-2 w-full">
            <TransfersPaginated ref="trRef"></TransfersPaginated>
        </div>

    </div>
  </main>
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