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
import SavingsAllocationsCreate from "../features/savings/SavingsAllocationsCreate.vue";
import SavingsCategories from "../features/savings/SavingsCategories.vue";
import type {SavingsGroup, SavingsStatistics} from "../../models/savings.ts";
import DisplayMonthlyDate from "../components/shared/DisplayMonthlyDate.vue";
import SavingsStatDisplay from "../features/savings/SavingsStatDisplay.vue";
import SavingsDeductionsCreate from "../features/savings/SavingsDeductionsCreate.vue";
import ActionRow from "../components/shared/ActionRow.vue";
import ActiveFilters from "../components/shared/filters/ActiveFilters.vue";
import {useBudgetStore} from "../../services/stores/budgetStore.ts";
import ReoccurringActionsDisplay from "../components/shared/ReoccurringActionsDisplay.vue";
import {useActionStore} from "../../services/stores/reoccurringActionStore.ts";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();
const budgetStore = useBudgetStore();
const actionStore = useActionStore();

const loadingSavings = ref(true);
const loadingGroupedSavings = ref(true);
const savings = ref([]);
const groupedSavings = ref<SavingsGroup[]>([]);

const addSavingsAllocationModal = ref(false);
const addSavingsDeductionModal = ref(false);
const addCategoryModal = ref(false);
const savingsStatistics = ref<SavingsStatistics[]>([]);

const dataCount = computed(() => {return savings.value.length});
const activeAllocation = computed(() => {return budgetStore.getAllocationByIndex("savings") || 0});

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
  { field: 'transaction_type', header: 'Type' },
  { field: 'savings_category', header: 'Category' },
  { field: 'adjusted_amount', header: 'Amount' },
  { field: 'transaction_date', header: 'Date' },
  { field: 'description', header: 'Description' },
]);

const savingsCategories = computed(() => savingsStore.savingsCategories);
const filteredSavingCategories = ref([]);

onMounted(async () => {
  await init();
  await actionStore.getAllActionsForCategory("savings_categories");
});

async function initData() {
  await getData();
  await getGroupedData();
  await savingsStore.getSavingsYears();
}

async function init() {
  await initData();
  await savingsStore.getSavingsCategories();
  sort.value = vueHelper.initSort();
}

async function getData(new_page = null) {

  loadingSavings.value = true;
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
    loadingSavings.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getGroupedData() {

  loadingGroupedSavings.value = true;
  if(dataCount.value < 1)
    return;

  try {

    let response = await savingsStore.getAllGroupedSavings(savingsStore.currentYear);
    groupedSavings.value = response.data;
    loadingGroupedSavings.value = false;
    calculateSavingsStatistics(
        groupedSavings.value,
        savingsStatistics,
        item => item.category_id,
        item => item.category_name,
        item => (item as any).goal_progress,
        item => (item as any).goal_target,
        item => (item as any).goal_spent,
    );
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function calculateSavingsStatistics<T>(
    groupedItems: T[],
    targetRef: {
      value: {
        category: string;
        goal_progress: number | null;
        goal_target: number | null;
        goal_spent: number | null;
        goal_remaining: number | null;
      }[]
    },
    getCategoryId: (item: T) => number,
    getCategoryName: (item: T) => string,
    getGoalProgress?: (item: T) => number | null,
    getGoalTarget?: (item: T) => number | null,
    getGoalSpent?: (item: T) => number | null,
): void {
  if (!groupedItems || groupedItems.length === 0) {
    return;
  }

  const filteredItems = groupedItems.filter(item => getCategoryName(item) !== "Total");

  const groupedData = filteredItems.reduce<Record<string, {
    categoryName: string;
    goalProgress: number;
    goalTarget: number;
    goalSpent: number;
  }>>((acc, curr) => {
    const categoryId = getCategoryId(curr);
    const key = `${categoryId}`;

    if (!acc[key]) {
      acc[key] = {
        categoryName: getCategoryName(curr),
        goalProgress: getGoalProgress ? getGoalProgress(curr) ?? 0 : 0,
        goalTarget: getGoalTarget ? getGoalTarget(curr) ?? 0 : 0,
        goalSpent: getGoalSpent ? getGoalSpent(curr) ?? 0 : 0,
      };
    } else {
      acc[key].goalProgress += getGoalProgress ? getGoalProgress(curr) ?? 0 : 0;
      acc[key].goalSpent += getGoalSpent ? getGoalSpent(curr) ?? 0 : 0;
    }

    return acc;
  }, {});

  const rows = Object.values(groupedData).map(group => {
    const goalRemaining = group.goalTarget - group.goalProgress;

    return {
      category: group.categoryName,
      goal_progress: group.goalProgress,
      goal_target: group.goalTarget,
      goal_spent: group.goalSpent,
      goal_remaining: goalRemaining,
    };
  });

  const totalRow = {
    category: "Total",
    goal_progress: rows.reduce((sum, row) => sum + (row.goal_progress ?? 0), 0),
    goal_target: rows.reduce((sum, row) => sum + (row.goal_target ?? 0), 0),
    goal_spent: rows.reduce((sum, row) => sum + (row.goal_spent ?? 0), 0),
    goal_remaining: rows.reduce((sum, row) => sum + ((row.goal_target ?? 0) - (row.goal_progress ?? 0)), 0),
  };

  targetRef.value = [totalRow, ...rows];
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'insertRecAction': {
      await actionStore.getAllActionsForCategory("savings_categories");
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

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-allocation': {
      addSavingsAllocationModal.value = value;
      break;
    }
    case 'add-deduction': {
      addSavingsDeductionModal.value = value;
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
    case "transaction_date": {
      filterType.value = "date";
      break;
    }
    case "adjusted_amount": {
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

onMounted(async () => {
  await budgetStore.getCurrentBudget();
})

provide("initData", initData);
provide("switchSort", switchSort);
provide("toggleFilterOverlay", toggleFilterOverlay);
provide('submitFilter', submitFilter);
provide('removeFilter', removeFilter);

</script>

<template>
  <Dialog v-model:visible="addSavingsAllocationModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Add allocation">
    <SavingsAllocationsCreate></SavingsAllocationsCreate>
  </Dialog>
  <Dialog v-model:visible="addSavingsDeductionModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Add deduction">
    <SavingsDeductionsCreate></SavingsDeductionsCreate>
  </Dialog>
  <Dialog v-model:visible="addCategoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Savings categories">
    <SavingsCategories @insertReoccurringActionEvent="handleEmit('insertRecAction')" :restricted="false" :availableAllocation="activeAllocation"></SavingsCategories>
  </Dialog>
  <Popover ref="filterOverlayRef">
    <BaseFilter :activeColumn="activeFilterColumn"
                :filter="filterObj" :filters="filters" :filterType="filterType"></BaseFilter>
  </Popover>

  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <ActionRow>
        <template #yearPicker>
          <YearPicker records="savings" :year="savingsStore.currentYear"
                      :availableYears="savingsStore.savingsYears"  @update:year="updateYear" />
        </template>
        <template #allocation>
            <div class="flex flex row w-full gap-1 align-items-center">
              <div class="flex-column">
                {{ "Method: " + activeAllocation?.method }}
              </div>
              <div class="flex-column" v-if="activeAllocation?.method === 'percentage'">
                {{ "Allocation: " + vueHelper.displayAsPercentage(activeAllocation?.allocation) }}
              </div>
              <div class="flex-column">
                {{ "Value: " + vueHelper.displayAsCurrency(activeAllocation?.allocated_value) }}
              </div>
            </div>
        </template>
        <template #activeFilters>
          <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />
        </template>
      </ActionRow>

      <div class="flex flex-row p-1">
        <h3>
          Manage entries
        </h3>
      </div>

      <div class="flex flex-row p-1 w-full gap-2">
        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Allocations</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-check" label="Add" @click="manipulateDialog('add-allocation', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Deductions</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file" label="Add" @click="manipulateDialog('add-deduction', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-arrow-up" label="Manage" @click="manipulateDialog('add-category', true)"></Button>
        </div>

      </div>

      <div class="flex flex-row p-1">
        <h3>
          Savings by month
        </h3>
      </div>


      <DisplayMonthlyDate :groupedValues="groupedSavings" :dataCount="dataCount"/>


      <div class="flex flex-row p-1 w-full">
        <h3>
          All savings
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingSavings" :value="savings" size="small"
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
              <template v-if="field === 'adjusted_amount'">
                {{ vueHelper.displayAsCurrency(data.adjusted_amount) }}
              </template>
              <template v-else-if="field === 'transaction_date'">
                {{ dateHelper.formatDate(data?.transaction_date, true) }}
              </template>
              <template v-else-if="field === 'savings_category'">
                {{ data[field]["name"] }}
              </template>
              <template v-else>
                {{ data[field] }}
              </template>
            </template>

            <template #editor="{ data, field }">
              <template v-if="field === 'adjusted_amount'">
                <InputNumber size="small" v-model="data[field]" mode="currency" currency="EUR" locale="de-DE" autofocus fluid />
              </template>
              <template v-else-if="field === 'transaction_date'">
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

      <SavingsStatDisplay :savingsStats="savingsStatistics" :dataCount="dataCount" />

      <div class="flex flex-row p-1">
        <h3>
          Reoccurring
        </h3>
      </div>

      <ReoccurringActionsDisplay categoryName="savings" :categoryItems="actionStore.reoccurringActions" />

    </div>
  </div>
</template>

<style scoped>

</style>