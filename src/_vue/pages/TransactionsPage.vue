<script setup lang="ts">
import AddTransaction from "../components/forms/AddTransaction.vue";
import {computed, onMounted, provide, ref} from "vue";
import type {Account} from "../../models/account_models.ts";
import vueHelper from "../../utils/vueHelper.ts";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import dateHelper from "../../utils/dateHelper.ts";
import LoadingSpinner from "../components/base/LoadingSpinner.vue";
import ActionRow from "../components/layout/ActionRow.vue";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import type {FilterObj} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import filterHelper from "../../utils/filterHelper.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const apiPrefix = "transactions";

const addTransactionModal = ref(false);

onMounted(async () => {
  await getData();
  await transactionStore.getCategories();
})

const loadingRecords = ref(true);
const records = ref<Account[]>([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
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


const activeColumns = ref([
  { field: 'account', header: 'Account' },
  { field: 'category', header: 'Category' },
  { field: 'amount', header: 'Amount' },
  { field: 'txn_date', header: 'Date' },
  { field: 'description', header: 'Description' },
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

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'addTransaction': {
      addTransactionModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'addTransaction': {
      addTransactionModal.value = false;
      await getData();
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

provide("initData", getData);
provide("switchSort", switchSort);

</script>

<template>

  <Dialog class="rounded-dialog" v-model:visible="addTransactionModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Add transaction">
    <AddTransaction entity="account" @addTransaction="handleEmit('addTransaction')"></AddTransaction>
  </Dialog>

  <Popover ref="filterOverlayRef">
    <div class="flex flex-column gap-2" style="width: 500px">
      <FilterMenu
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
            <div class="hover flex flex-row align-items-center gap-2" @click="toggleFilterOverlay($event)"
                 style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)">
              <i class="pi pi-filter" style="font-size: 0.845rem"></i>
              <div>Filter</div>
            </div>
          </template>
        </ActionRow>
      </div>

      <div class="flex flex-row gap-2 w-full" >
        <DataTable class="w-full  enhanced-table" dataKey="id" :loading="loadingRecords" :value="records">
          <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
          <template #loading> <LoadingSpinner></LoadingSpinner> </template>
          <template #footer>
            <Paginator v-model:first="paginator.from"
                       v-model:rows="paginator.rowsPerPage"
                       :rowsPerPageOptions="rows"
                       :totalRecords="paginator.total"
                       @page="onPage($event)">
              <template #end>
                <div>
                  {{
                    "Showing " + paginator.from + " to " + paginator.to + " out of " + paginator.total + " " + "records"
                  }}
                </div>
              </template>
            </Paginator>
          </template>

          <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
            <template #header >
              <ColumnHeader  :header="col.header" :field="col.field" :sort="sort"></ColumnHeader>
            </template>
            <template #body="{ data, field }">
              <template v-if="field === 'amount'">
                {{ vueHelper.displayAsCurrency(data.amount) }}
              </template>
              <template v-else-if="field === 'txn_date'">
                {{ dateHelper.formatDate(data?.txn_date, true) }}
              </template>
              <template v-else-if="field === 'category' || field === 'account'">
                {{ data[field]["name"] }}
              </template>
              <template v-else>
                {{ data[field] }}
              </template>
            </template>
          </Column>

        </DataTable>
      </div>

    </div>
  </main>
</template>

<style scoped>

</style>