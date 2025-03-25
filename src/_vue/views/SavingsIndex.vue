<script setup lang="ts">
import {useToastStore} from "../../services/stores/toastStore.ts";
import {computed, onMounted, provide, ref} from "vue";
import type {Statistics} from "../../models/shared.ts";
import vueHelper from "../../utils/vueHelper.ts";
import {useSavingsStore} from "../../services/stores/savingsStore.ts";

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
    {{ "Savings" }}
</template>

<style scoped>

</style>