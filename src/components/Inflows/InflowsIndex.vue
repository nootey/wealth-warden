<script setup lang="ts">
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {computed, ref} from "vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {useToastStore} from "../../services/stores/toastStore.ts";
import dateHelper from "../../utils/dateHelper.ts"
import ValidationError from "../Validation/ValidationError.vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import InflowCategories from "./InflowCategories.vue";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const loading = ref(true);
const inflows = ref([]);
const newInflow = ref(initInflow());
const inflowCategories = ref([]);
const filteredInflowCategories = ref([]);

const inflowRules = {
  newInflow: {
    amount: {
      required,
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
  await getInflowCategories();
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

  loading.value = true;
  if(new_page)
    page.value = new_page;

  try {

    let paginationResponse = await inflowStore.getInflowsPaginated(params.value, page.value);
    inflows.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loading.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

async function getInflowCategories() {
  try {
    inflowCategories.value = await inflowStore.getInflowCategories();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
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
    toastStore.successResponseToast(response);
    await getData();
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


</script>

<template>
  <div class="flex w-12 p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1">
        <h1>
          Inflows
        </h1>
      </div>

      <div class="flex flex-row gap-2">

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

      </div>

      <div class="flex flex-row gap-2" style="border-top: 1px solid var(--text-primary)">
        <DataTable dataKey="id" :loading="loading" :value="inflows" class="p-datatable-sm">
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
          <Column field="amount" header="Amount"></Column>
          <Column field="inflow_date" header="Date">
            <template #body="slotProps">
               {{ dateHelper.formatDate(slotProps.data?.inflow_date, true) }}
            </template>
          </Column>

        </DataTable>
      </div>

    </div>

    <InflowCategories></InflowCategories>


  </div>
</template>

<style scoped>

</style>