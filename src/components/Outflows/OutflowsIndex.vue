<script setup lang="ts">
import {useToastStore} from "../../services/stores/toastStore.ts";
import {useActionStore} from "../../services/stores/reoccurringActionStore.ts";
import {computed, onMounted, provide, ref} from "vue";
import type {OutflowGroup} from "../../models/outflows.ts";
import type {GroupedItem, Statistics} from "../../models/shared.ts";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";
import vueHelper from "../../utils/vueHelper.ts";
import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../Validation/ValidationError.vue";
import ReoccurringActionsDisplay from "../Shared/ReoccurringActionsDisplay.vue";
import BasicStatDisplay from "../Shared/BasicStatDisplay.vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import DisplayMonthlyDate from "../Shared/DisplayMonthlyDate.vue";

const outflowStore = useOutflowStore();
const toastStore = useToastStore();
const actionStore = useActionStore();

const loadingOutflows = ref(true);
const loadingGroupedOutflows = ref(true);
const outflows = ref([]);
const groupedOutflows = ref<OutflowGroup[]>([]);

const addOutflowModal = ref(false);
const addCategoryModal = ref(false);
const outflowStatistics = ref<Statistics[]>([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: [],
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

onMounted(async () => {
  await getData();
  await outflowStore.getOutflowCategories();
  await actionStore.getAllActionsForCategory("outflow");
  await getGroupedData();
  sort.value = vueHelper.initSort();
});

async function getData(new_page = null) {

  loadingOutflows.value = true;
  if(new_page)
    page.value = new_page;

  try {

    let paginationResponse = await outflowStore.getOutflowsPaginated(params.value, page.value);
    outflows.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingOutflows.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getGroupedData() {

  loadingGroupedOutflows.value = true;

  try {

    let response = await outflowStore.getAllGroupedOutflows();
    groupedOutflows.value = response.data;
    loadingGroupedOutflows.value = false;
    calculateStatistics(groupedOutflows.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}


async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

async function removeOutflow(id: number) {
  try {
    let response = await outflowStore.deleteOutflow(id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-outflow': {
      addOutflowModal.value = value;
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

function calculateStatistics(groupedOutflows: OUtflowStat[]): void {

  if (!groupedOutflows || groupedOutflows.length === 0) {
    return;
  }

  const groupedData = groupedOutflows.reduce<Record<number, GroupedItem>>((acc, curr) => {
    const { outflow_category_id, outflow_category_name, total_amount, month } = curr;

    // Initialize the group if it doesn't exist
    if (!acc[outflow_category_id]) {
      acc[outflow_category_id] = {
        categoryName: outflow_category_name,
        total: 0,
        months: new Set<number>(),
      };
    }

    // Add the amount and record the month
    acc[outflow_category_id].total += total_amount;
    acc[outflow_category_id].months.add(month);

    return acc;
  }, {});

  outflowStatistics.value = Object.values(groupedData).map((category: GroupedItem) => {
    const monthCount = category.months.size;
    return {
      category: category.categoryName,
      total: category.total,
      average: category.total / monthCount
    };
  });
}

provide("getData", getData)
provide("getGroupedData", getGroupedData)

</script>

<template>
  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1">
        <h1>
          Outflows
        </h1>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Add a new item
        </h3>
      </div>


      <div class="flex flex-row p-1 w-full gap-3">
        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Outflows</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-box" label="View" @click="manipulateDialog('add-outflow', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Outflow categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-box" label="View" @click="manipulateDialog('add-category', true)"></Button>
        </div>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Outflows by month
        </h3>
      </div>

      <DisplayMonthlyDate :groupedValues="groupedOutflows" />

      <div class="flex flex-row p-1 w-full">
        <h3>
          All outflows
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingOutflows" :value="outflows" size="small">
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
                <i class="pi pi-pencil hover_icon"
                   @click="editOutflow(slotProps.data?.id)"></i>
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeOutflow(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column field="outflow_category.name" header="Category"></Column>
          <Column field="amount" header="Amount">
            <template #body="slotProps">
              {{ vueHelper.displayAsCurrency(slotProps.data.amount)}}
            </template>
          </Column>
          <Column field="outflow_date" header="Date">
            <template #body="slotProps">
              {{ dateHelper.formatDate(slotProps.data?.outflow_date, true) }}
            </template>
          </Column>

        </DataTable>
      </div>
    </div>

    <div class="flex flex-column w-3 p-2 gap-3" style="border-left: 1px solid var(--text-primary);">

      <div class="flex flex-row p-1">
        <h1>
          Statistics
        </h1>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Outflows
        </h3>
      </div>

      <BasicStatDisplay :basicStats="outflowStatistics" />

      <div class="flex flex-row p-1">
        <h3>
          Reoccurring
        </h3>
      </div>

      <ReoccurringActionsDisplay :categoryItems="actionStore.reoccurringActions" />
    </div>
  </div>
</template>

<style scoped>

</style>