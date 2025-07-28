<script setup lang="ts">
import {computed, onMounted, provide, ref} from "vue";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import dateHelper from "../../../utils/dateHelper.ts"
import vueHelper from "../../../utils/vueHelper.ts";
import ColumnHeader from "../../components/base/ColumnHeader.vue";
import BaseFilter from "../../components/filters/BaseFilter.vue";
import ActionRow from "../../components/layout/ActionRow.vue";
import ActiveFilters from "../../components/filters/ActiveFilters.vue";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";

const toast_store = useToastStore();
const transaction_store = useTransactionStore();

const loading_records = ref(true);
const records = ref([]);

const data_count = computed(() => {return records.value.length});

const activeFilers = ref([]);
const filterStorageIndex = ref("transaction-filters");
const filterObj = ref({});
const filterType = ref("text");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value)) ?? []);
const activeFilterColumn = ref(null)
const filterOverlayRef = ref(null);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
  }
});
const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(vueHelper.initSort());

const inflowColumns = ref([
  { field: 'inflow_category', header: 'Category' },
  { field: 'amount', header: 'Amount' },
  { field: 'inflow_date', header: 'Date' },
  { field: 'description', header: 'Description' }
]);


onMounted(async () => {
  await init();
});

async function initData() {
  await getData();
}

async function init() {
  await initData();
  sort.value = vueHelper.initSort();
}

async function getData(new_page = null) {

  loading_records.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await transaction_store.getTransactionsPaginated(
        { ...params.value, year: transaction_store.currentYear },
        page.value
    );
    records.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loading_records.value = false;
  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

async function removeInflow(id: number) {
  try {
    toast_store.successResponseToast(response);
    await initData();
  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-inflow': {
      addInflowModal.value = value;
      break;
    }
    case 'add-category': {
      addCategoryModal.value = value;
      break;
    }
    default: {
      break;
    }
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

function submitFilter(parameter) {
  if(!filterObj.value.value){
    vueHelper.formatInfoToast("Invalid value", "Input a filter value");
    return;
  }

  filterObj.value.parameter = parameter;
  addFilter(filterObj.value);
  getData();
}

function addFilter(filter, alternate = null) {
  let new_filter = {
    parameter: filter.parameter,
    operator: filter.operator,
    value: filterType === "text" ? filter.value.trim().replace(/\s+/g, " ") : filter.value,
  };

  let exists = filters.value.find((object) => {
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

function clearFilters(){
  filters.value.splice(0);
  localStorage.removeItem(filterStorageIndex.value);
  getData();
}

function removeFilter(index){
  filters.value.splice(index, 1);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value))
  getData();
}

function switchSort(column) {
  if (sort.value.field === column) {
    sort.value.order = vueHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
}

function toggleFilterOverlay(event, column) {

  switch (column) {
    case "inflow_date": {
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

provide("initData", initData);
provide("switchSort", switchSort);
provide("toggleFilterOverlay", toggleFilterOverlay);
provide('submitFilter', submitFilter);
provide('removeFilter', removeFilter);

</script>

<template>

<!--  <Dialog v-model:visible="addInflowModal" :breakpoints="{'801px': '90vw'}"-->
<!--          :modal="true" :style="{width: '800px'}" header="Add inflow">-->
<!--    <InflowCreate @insertReoccurringActionEvent="handleEmit('insertRecAction')"></InflowCreate>-->
<!--  </Dialog>-->
<!--  <Dialog v-model:visible="addCategoryModal" :breakpoints="{'801px': '90vw'}"-->
<!--          :modal="true" :style="{width: '800px'}" header="Inflow categories">-->
<!--    <InflowCategories :restricted="false"></InflowCategories>-->
<!--  </Dialog>-->
  <Popover ref="filterOverlayRef">
    <BaseFilter :activeColumn="activeFilterColumn"
                :filter="filterObj" :filters="filters" :filterType="filterType"></BaseFilter>
  </Popover>

  <div class="flex w-full p-2">

    <div class="flex w-full flex-column p-2 gap-3">

      <div class="flex flex-row p-1 w-full">
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

      <div class="flex flex-row gap-2 w-full">
          <DataTable class="w-full" dataKey="id" :loading="loading_records" :value="records" size="small">
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

            <Column v-for="col of inflowColumns" :key="col.field" :field="col.field" style="width: 25%">
              <template #header>
                <ColumnHeader :header="col.header" :field="col.field" :sort="sort" :filter="true" :filters="filters"></ColumnHeader>
              </template>
              <template #body="{ data, field }">
                <template v-if="field === 'amount'">
                  {{ vueHelper.displayAsCurrency(data.amount) }}
                </template>
                <template v-else-if="field === 'inflow_date'">
                  {{ dateHelper.formatDate(data?.inflow_date, true) }}
                </template>
                <template v-else-if="field === 'inflow_category'">
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
  </div>
</template>

<style scoped>

</style>