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
import FilterMenu from "../components/filters/FilterMenu.vue";

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
const rows = ref([25, 50, 100]);
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
const filterStorageIndex = ref(apiPrefix+"-filters");
const filterObj = ref<FilterObj>({
  source: apiPrefix,
  field: null,
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

function initFilter() {
  filterObj.value = {
    source: apiPrefix,
    field: null,
    operator: 'like',
    value: null
  }
}

function submitFilter(field: string) {
  if(!filterObj.value.value){
    vueHelper.formatInfoToast("Invalid value", "Input a filter value");
    return;
  }

  filterObj.value.field = field;
  addFilter(filterObj.value);
  getData();
}

function addFilter(filter: FilterObj, alternate = null) {
  let new_filter = {
    source: apiPrefix,
    field: filter.field,
    operator: filter.operator,
    value: filterType.value === "text" ? filter.value.trim().replace(/\s+/g, " ") : filter.value,
  };

  let exists = filters.value.find((object: FilterObj) => {
    // Compare only the relevant properties
    return (
        object.field === new_filter.field &&
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
    <FilterMenu :columns="activeColumns" :apiSource="apiPrefix"
                @apply="(list) => { filters = list; localStorage.setItem(filterStorageIndex, JSON.stringify(list)); getData(); filterOverlayRef.hide(); }"
                @clear="() => { filters = []; localStorage.removeItem(filterStorageIndex); getData(); }"
                @cancel="() => filterOverlayRef.hide()"></FilterMenu>
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
<!--          <template #activeFilters>-->
<!--            <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />-->
<!--          </template>-->
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

          <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
            <template #header>
              <ColumnHeader :header="col.header" :field="col.field" :sort="sort"></ColumnHeader>
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