<script setup lang="ts">
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {computed, ref} from "vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {useToastStore} from "../../services/stores/toastStore.ts";
import dateHelper from "../../utils/dateHelper.ts"
import ValidationError from "../Validation/ValidationError.vue";
import {required, numeric,} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import InflowCategories from "./InflowCategories.vue";
import vueHelper from "../../utils/vueHelper.ts";
import type {GroupedItem, Inflow, Statistics, InflowGroup} from '../../models/inflows.ts';
import BasicStatDisplay from "../Shared/BasicStatDisplay.vue";
import DisplayMonthlyDate from "../Shared/DisplayMonthlyDate.vue";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const loadingInflows = ref(true);
const loadingGroupedInflows = ref(true);
const inflows = ref([]);
const groupedInflows = ref<InflowGroup[]>([]);
const newInflow = ref(initInflow());

const inflowCategories = computed(() => inflowStore.inflowCategories);
const filteredInflowCategories = ref([]);
const categoryModal = ref(false);
const inflowStatistics = ref<Statistics[]>([]);

const inflowRules = {
  newInflow: {
    amount: {
      required,
      numeric,
      minValue: 0,
      maxValue: 1000000000,
      $autoDirty: true
    },
    inflowCategory: {
      required,
      $autoDirty: true
    },
    inflowDate: {
      required,
      $autoDirty: true
    },
  }
};

const v$ = useVuelidate(inflowRules, { newInflow });

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
const sort = ref(initSort(true));

init();

async function init() {
  await getData();
  await inflowStore.getInflowCategories();
  await getGroupedData();
  initSort();
}

function initSort(init = false) {
  let obj = {
    order: -1,
    field: 'created_at'
  };
  if (init) {
    return obj;
  }
  sort.value = obj;
}

function initInflow():object {
  return {
    amount: null,
    inflowCategory: [],
    inflowDate: dateHelper.formatDate(new Date(), true),
  }
}

const searchInflowCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredInflowCategories.value = [...inflowCategories.value];
    } else {
      filteredInflowCategories.value = inflowCategories.value.filter((inflowCategory) => {
        return inflowCategory.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function getData(new_page = null) {

  loadingInflows.value = true;
  if(new_page)
    page.value = new_page;

  try {

    let paginationResponse = await inflowStore.getInflowsPaginated(params.value, page.value);
    inflows.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingInflows.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getGroupedData() {

  loadingGroupedInflows.value = true;

  try {

    let response = await inflowStore.getAllGroupedInflows();
    groupedInflows.value = response.data;
    loadingGroupedInflows.value = false;
    calculateStatistics(groupedInflows.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

async function createNewInflow() {

  v$.value.newInflow.amount.$touch();
  v$.value.newInflow.inflowDate.$touch();
  v$.value.newInflow.inflowCategory.$touch();
  if (v$.value.newInflow.$error) return;

  try {
    let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");
    let response = await inflowStore.createInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date});

    newInflow.value = initInflow();
    v$.value.newInflow.$reset();

    await getData();
    await getGroupedData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function editInflow(id: number) {
  console.log(id)
}

async function removeInflow(id: number) {
  try {
    let response = await inflowStore.deleteInflow(id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'category': {
      categoryModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

function calculateStatistics(groupedInflows: Inflow[]): void {

  if (!groupedInflows || groupedInflows.length === 0) {
    return;
  }

  const groupedData = groupedInflows.reduce<Record<number, GroupedItem>>((acc, curr) => {
    const { inflow_category_id, inflow_category_name, total_amount, month } = curr;

    // Initialize the group if it doesn't exist
    if (!acc[inflow_category_id]) {
      acc[inflow_category_id] = {
        categoryName: inflow_category_name,
        total: 0,
        months: new Set<number>(),
      };
    }

    // Add the amount and record the month
    acc[inflow_category_id].total += total_amount;
    acc[inflow_category_id].months.add(month);

    return acc;
  }, {});

  inflowStatistics.value = Object.values(groupedData).map((category: GroupedItem) => {
    const monthCount = category.months.size;
    return {
      category: category.categoryName,
      total: category.total,
      average: category.total / monthCount
    };
  });
}

</script>

<template>
  <Dialog v-model:visible="categoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Inflow categories">
    <InflowCategories></InflowCategories>
  </Dialog>
  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1">
        <h1>
          Inflows
        </h1>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Add a new inflow
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">

        <div class="flex flex-column">
          <ValidationError :isRequired="true" :message="v$.newInflow.inflowCategory.$errors[0]?.$message">
            <label>Category</label>
          </ValidationError>
          <InputGroup>
            <InputGroupAddon>
              <i class="pi pi-address-book"></i>
            </InputGroupAddon>
            <AutoComplete size="small" v-model="newInflow.inflowCategory" :suggestions="filteredInflowCategories"
                          @complete="searchInflowCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
          </InputGroup>
        </div>

        <div class="flex flex-column">
          <ValidationError :isRequired="true" :message="v$.newInflow.amount.$errors[0]?.$message">
            <label>Amount</label>
          </ValidationError>
          <InputGroup>
            <InputGroupAddon>
              <i class="pi pi-wallet"></i>
            </InputGroupAddon>
            <InputNumber size="small" v-model="newInflow.amount" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
          </InputGroup>
        </div>

        <div class="flex flex-column">
          <ValidationError :isRequired="true" :message="v$.newInflow.inflowDate.$errors[0]?.$message">
            <label>Date</label>
          </ValidationError>
          <DatePicker v-model="newInflow.inflowDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                      style="height: 42px;"/>
        </div>

        <div class="flex flex-column">
          <ValidationError :isRequired="false" message="">
            <label>Submit</label>
          </ValidationError>
          <Button icon="pi pi-cart-plus" @click="createNewInflow" style="height: 42px;" />
        </div>

        <div class="flex flex-column" style="margin-left: auto;">
          <ValidationError :isRequired="false" message="">
            <label>Inflow categories</label>
          </ValidationError>
          <Button icon="pi pi-box" label="View" @click="manipulateDialog('category', true)"></Button>
        </div>

      </div>

      <div class="flex flex-row p-1">
        <h3>
          Inflows by month
        </h3>
      </div>

      <DisplayMonthlyDate :groupedValues="groupedInflows" />

      <div class="flex flex-row p-1 w-full">
        <h3>
          All inflows
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingInflows" :value="inflows" size="small">
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
                   @click="editInflow(slotProps.data?.id)"></i>
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeInflow(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column field="inflow_category.name" header="Category"></Column>
          <Column field="amount" header="Amount">
            <template #body="slotProps">
              {{ vueHelper.displayAsCurrency(slotProps.data.amount)}}
            </template>
          </Column>
          <Column field="inflow_date" header="Date">
            <template #body="slotProps">
               {{ dateHelper.formatDate(slotProps.data?.inflow_date, true) }}
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
          Inflows
        </h3>
      </div>

      <BasicStatDisplay :basicStats="inflowStatistics" />

      <div class="flex flex-row p-1">
        <h3>
          Reoccurring
        </h3>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>