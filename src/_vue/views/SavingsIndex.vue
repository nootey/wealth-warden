<script setup lang="ts">
import {useToastStore} from "../../services/stores/toastStore.ts";
import {computed, onMounted, provide, ref} from "vue";
import vueHelper from "../../utils/vueHelper.ts";
import {useSavingsStore} from "../../services/stores/savingsStore.ts";
import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../components/validation/ValidationError.vue";
import LoadingSpinner from "../components/ui/LoadingSpinner.vue";
import ColumnHeader from "../components/shared/ColumnHeader.vue";
import YearPicker from "../components/shared/YearPicker.vue";
import BaseFilter from "../components/shared/filters/BaseFilter.vue";
import SavingsCreate from "../features/savings/SavingsCreate.vue";
import SavingsCategories from "../features/savings/SavingsCategories.vue";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();

const loadingSavingss = ref(true);
const savings = ref([]);

const addSavingsModal = ref(false);
const addCategoryModal = ref(false);


const dataCount = computed(() => {return savings.value.length});

const activeFilers = ref([]);
const filterStorageIndex = ref("savings-filters");
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

const savingsColumns = ref([
  { field: 'savings_category', header: 'Category' },
  { field: 'savings_date', header: 'Date' },
]);

const savingsCategories = computed(() => savingsStore.savingsCategories);
const filteredSavingCategories = ref([]);

onMounted(async () => {
  await init();
});

async function initData() {
  await getData();
  await savingsStore.getSavingsYears();
}

async function init() {
  await initData();
  await savingsStore.getSavingsCategories();
  sort.value = vueHelper.initSort();
}

async function getData(new_page = null) {

  loadingSavingss.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await savingsStore.getSavingsPaginated(
        { ...params.value, year: savingsStore.currentYear },
        page.value
    );
    savings.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingSavingss.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-savings': {
      addSavingsModal.value = value;
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

const searchSavingsCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredSavingCategories.value = [...savingsCategories.value];
    } else {
      filteredSavingCategories.value = savingsCategories.value.filter((category) => {
        return category.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function updateYear(newYear: number) {
  savingsStore.currentYear = newYear;
  await init();
}

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
    case "savings_date": {
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
  <Dialog v-model:visible="addSavingsModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Add savings">
    <SavingsCreate></SavingsCreate>
  </Dialog>
  <Dialog v-model:visible="addCategoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Savings categories">
    <SavingsCategories :restricted="false"></SavingsCategories>
  </Dialog>
  <Popover ref="filterOverlayRef">
    <BaseFilter :activeColumn="activeFilterColumn"
                :filter="filterObj" :filters="filters" :filterType="filterType"></BaseFilter>
  </Popover>

  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1 fap-2 align-items-center">
        <div class="flex flex-column p-1">
          Select year:
        </div>
        <div>
          <YearPicker records="savings" :year="savingsStore.currentYear"
                      :availableYears="savingsStore.savingsYears"  @update:year="updateYear" />
        </div>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Manage entries
        </h3>
      </div>

      <div class="flex flex-row p-1 w-full gap-2">
        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Savings</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-check" label="Create" @click="manipulateDialog('add-savings', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Savings categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-arrow-up" label="Manage" @click="manipulateDialog('add-category', true)"></Button>
        </div>

      </div>


      <div class="flex flex-row p-1 w-full">
        <h3>
          All savings
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingSavingss" :value="savings" size="small"
                   editMode="cell">
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
                   @click="removeSaving(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column v-for="col of savingsColumns" :key="col.field" :field="col.field" style="width: 25%">
            <template #header>
              <ColumnHeader :header="col.header" :field="col.field" :sort="sort" :filter="true" :filters="filters"></ColumnHeader>
            </template>
            <template #body="{ data, field }">
              <template v-if="field === 'amount'">
                {{ vueHelper.displayAsCurrency(data.amount) }}
              </template>
              <template v-else-if="field === 'savings_date'">
                {{ dateHelper.formatDate(data?.savings_date, true) }}
              </template>
              <template v-else-if="field === 'savings_category'">
                {{ data[field]["name"] }}
              </template>
              <template v-else>
                {{ data[field] }}
              </template>
            </template>

            <template #editor="{ data, field }">
              <template v-if="field === 'amount'">
                <InputNumber size="small" v-model="data[field]" mode="currency" currency="EUR" locale="de-DE" autofocus fluid />
              </template>
              <template v-else-if="field === 'savings_date'">
                <DatePicker v-model="data[field]" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                            style="height: 42px;"/>
              </template>
              <template v-else-if="field === 'savings_category'">
                <AutoComplete size="small" v-model="data[field]" :suggestions="filteredSavingCategories"
                              @complete="searchSavingsCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
              </template>
              <template v-else>
                <InputText size="small" v-model="data[field]" autofocus fluid />
              </template>
            </template>
          </Column>

        </DataTable>
      </div>
    </div>

    <div class="flex flex-column w-3 p-2 gap-3" style="border-left: 1px solid var(--text-primary);">

      <div class="flex flex-row p-1">
        <h2>
          Statistics
        </h2>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Savings
        </h3>
      </div>

<!--      <BasicStatDisplay :basicStats="savingsStatistics" :limit="false" :dataCount="dataCount" />-->

    </div>
  </div>
</template>

<style scoped>

</style>