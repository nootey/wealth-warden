<script setup lang="ts">
import TransactionForm from "../components/forms/TransactionForm.vue";
import {computed, onMounted, provide, ref} from "vue";
import type {Transaction} from "../../models/transaction_models.ts";
import vueHelper from "../../utils/vue_helper.ts";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import dateHelper from "../../utils/date_helper.ts";
import LoadingSpinner from "../components/base/LoadingSpinner.vue";
import ActionRow from "../components/layout/ActionRow.vue";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import type {FilterObj} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import filterHelper from "../../utils/filter_helper.ts";
import type {Category} from "../../models/transaction_models.ts";
import type {Column} from "../../services/filter_registry.ts";
import {useConfirm} from "primevue/useconfirm";
import {useAccountStore} from "../../services/stores/account_store.ts";
import type {Account} from "../../models/account_models.ts";
import TransfersPaginated from "../features/TransfersPaginated.vue";
import CustomPaginator from "../components/base/CustomPaginator.vue";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

const confirm = useConfirm();

const apiPrefix = "transactions";

const transfersPaginatedRef = ref<InstanceType<typeof TransfersPaginated> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const updateTransactionID = ref(null);
const includeDeleted = ref(false);

onMounted(async () => {
  await getData();
  await transactionStore.getCategories();
  await accountStore.getAllAccounts();
})

const loadingRecords = ref(true);
const records = ref<Transaction[]>([]);
const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
    include_deleted: includeDeleted.value,
  }
});
const rows = ref([25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value
});
const page = ref(1);
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

async function getData(new_page = null) {

  loadingRecords.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await sharedStore.getRecordsPaginated(
        apiPrefix,
        { ...params.value },
        page.value
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
    case 'completeOperation': {
      createModal.value = false;
      updateModal.value = false;
      await getData();
      await transfersPaginatedRef.value?.getData();
      break;
    }
    default: {
      break;
    }
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

function applyFilters(list: FilterObj[]){
  filters.value = filterHelper.mergeFilters(filters.value, list);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  getData();
  filterOverlayRef.value.hide();
}

function clearFilters(){
  filters.value = [];
  localStorage.removeItem(filterStorageIndex.value);
  cancelFilters();
  getData();
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

  getData();
}

function switchSort(column:string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
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
    await getData();
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
    <TransactionForm mode="create" @completeOperation="handleEmit('completeOperation')"></TransactionForm>
  </Dialog>

  <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal" :breakpoints="{'501px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Update transaction">
    <TransactionForm mode="update" :recordId="updateTransactionID" @completeOperation="handleEmit('completeOperation')"></TransactionForm>
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

    <div class="flex flex-row justify-content-between align-items-center p-3 w-full"
         style="border-top-right-radius: 8px; border-top-left-radius: 8px;
         border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px;">

      <div style="font-weight: bold;">Transactions</div>
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
                <div class="hover_icon flex flex-row align-items-center gap-2" @click="toggleFilterOverlay($event)"
                     style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)">
                  <i class="pi pi-filter" style="font-size: 0.845rem"></i>
                  <div>Filter</div>
                </div>
              </template>
            <template #includeDeleted>
                <div class="flex align-items-center gap-2" style="margin-left: auto;">
                    <span style="font-size: 0.8rem;">Deleted</span>
                    <ToggleSwitch v-model="includeDeleted" @update:modelValue="getData()" />
                </div>
            </template>
        </ActionRow>
      </div>

        <div class="flex flex-row gap-2 w-full">
            <DataTable class="w-full enhanced-table" dataKey="id" :loading="loadingRecords" :value="records">
              <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
              <template #loading> <LoadingSpinner></LoadingSpinner> </template>
              <template #footer>
                  <CustomPaginator :paginator="paginator" :rows="rows" @onPage="onPage"/>
              </template>

              <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
                <template #header >
                  <ColumnHeader  :header="col.header" :field="col.field" :sort="sort"></ColumnHeader>
                </template>
                <template #body="{ data, field }">
                  <template v-if="field === 'amount'">
                    {{ vueHelper.displayAsCurrency(data.transaction_type == "expense" ? (data.amount*-1) : data.amount) }}
                  </template>
                  <template v-else-if="field === 'txn_date'">
                    {{ dateHelper.formatDate(data?.txn_date, true) }}
                  </template>
                  <template v-else-if="field === 'account'">
                    <span class="hover" @click="manipulateDialog('updateTransaction', data['id'])">
                      {{ data[field]["name"] }}
                    </span>
                  </template>
                  <template v-else-if="field === 'category'">
                    {{ data[field]["name"] }}
                  </template>
                  <template v-else>
                    {{ data[field] }}
                  </template>
                </template>
              </Column>

              <Column header="Actions">
                <template #body="slotProps">
                  <i class="pi pi-trash hover_icon" style="font-size: 0.875rem; color: var(--p-red-300);"
                     @click="deleteConfirmation(slotProps.data?.id, slotProps.data.transaction_type)"></i>
                </template>
              </Column>

            </DataTable>
        </div>

        <label>Transfers</label>
        <div class="flex flex-row gap-2 w-full">
            <TransfersPaginated ref="transfersPaginatedRef"></TransfersPaginated>
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