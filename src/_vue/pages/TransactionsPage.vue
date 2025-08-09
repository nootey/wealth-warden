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
import BaseFilter from "../components/filters/BaseFilter.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import type {FilterObj} from "../../models/shared_models.ts";

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
    filters: [],
  }
});
const rows = ref([100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(vueHelper.initSort());

// const data_count = computed(() => {return records.value.length});
//
// const activeFilers = ref([]);
const filterStorageIndex = ref("transaction-filters");
const filterObj = ref<FilterObj>({
  parameter: null,
  operator: 'like',
  value: null
});
const filterType = ref("text");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"));
const activeFilterColumn = ref<String|null>(null)
const filterOverlayRef = ref<any>(null);


const activeColumns = ref([
  { field: 'account', header: 'Account' },
  { field: 'category', header: 'Category' },
  { field: 'amount', header: 'Amount' },
  { field: 'txn_date', header: 'Date' },
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

async function removeInflow(id: number) {
  try {
    console.log(id)
    // toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

// async function updateYear(newYear: number) {
//   inflowStore.currentYear = newYear;
//   await init();
// }

function initFilter() {
  filterObj.value = {
    parameter: null,
    operator: 'like',
    value: null
  }
}

function submitFilter(parameter: string) {
  if(!filterObj.value.value){
    vueHelper.formatInfoToast("Invalid value", "Input a filter value");
    return;
  }

  filterObj.value.parameter = parameter;
  addFilter(filterObj.value);
  getData();
}

function addFilter(filter: FilterObj, alternate = null) {
  let new_filter = {
    parameter: filter.parameter,
    operator: filter.operator,
    value: filterType.value === "text" ? filter.value.trim().replace(/\s+/g, " ") : filter.value,
  };

  let exists = filters.value.find((object: FilterObj) => {
    // Compare only the relevant properties
    return (
        object.parameter === new_filter.parameter &&
        object.operator === new_filter.operator &&
        object.value === new_filter.value
    );
  });

  if (exists === undefined) {
    filters.value.push(new_filter);
    localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value))
    if (!alternate) initFilter();
    filterOverlayRef.value.hide();
  }
}

// function clearFilters(){
//   filters.value.splice(0);
//   localStorage.removeItem(filterStorageIndex.value);
//   getData();
// }

function removeFilter(index: number){
  filters.value.splice(index, 1);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value))
  getData();
}

function switchSort(column:string) {
  if (sort.value.field === column) {
    sort.value.order = vueHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
}

function toggleFilterOverlay(event: any, column: string) {

  switch (column) {
    case "txn_date": {
      filterType.value = "date";
      break;
    }
    case "amount": {
      filterType.value = "number";
      break;
    }
    default: {
      filterType.value = "text";
      break;
    }
  }

  activeFilterColumn.value = column;
  filterOverlayRef.value.toggle(event);
}

provide("initData", getData);
provide("switchSort", switchSort);
provide("toggleFilterOverlay", toggleFilterOverlay);
provide('submitFilter', submitFilter);
provide('removeFilter', removeFilter);


</script>

<template>

  <Dialog class="rounded-dialog" v-model:visible="addTransactionModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Add transaction">
    <AddTransaction entity="account" @addTransaction="handleEmit('addTransaction')"></AddTransaction>
  </Dialog>

  <Popover ref="filterOverlayRef">
    <BaseFilter :activeColumn="activeFilterColumn"
                :filter="filterObj" :filters="filters" :filterType="filterType"></BaseFilter>
  </Popover>

  <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

    <div class="flex flex-row justify-content-between align-items-center p-3 w-full"
         style="border-top-right-radius: 8px; border-top-left-radius: 8px;
         border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px;">

      <div style="font-weight: bold;">Transactions</div>
      <Button label="New transaction" icon="pi pi-plus" @click="manipulateDialog('addTransaction', true)"
              style="background-color: var(--text-primary); color: var(--background-primary);
                border: none; border-radius: 6px; font-size: 0.875rem; padding: 0.5rem 1rem;"></Button>
    </div>

    <div class="flex flex-column justify-content-center p-3 w-full gap-3"
         style="border-bottom-right-radius: 8px; border-bottom-left-radius: 8px;
         border: 1px solid var(--border-color); background: var(--background-secondary);
         max-width: 1000px;">

      <div class="flex flex-row w-full">
        <ActionRow>
          <template #yearPicker>
            <!--          <YearPicker records="inflows" :year="transaction_store.currentYear"-->
            <!--                      :availableYears="transaction_store.inflowYears"  @update:year="updateYear" />-->
          </template>
          <template #activeFilters>
            <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />
          </template>
        </ActionRow>
      </div>

      <div class="flex flex-row gap-2 w-full" >
        <DataTable class="w-full enhanced-table" dataKey="id" :loading="loadingRecords" :value="records" size="small">
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
          <Column header="Actions">
            <template #body="slotProps">
              <div class="flex flex-row align-items-center gap-2">
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeInflow(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
            <template #header>
              <ColumnHeader :header="col.header" :field="col.field" :sort="sort" :filter="true" :filters="filters"></ColumnHeader>
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