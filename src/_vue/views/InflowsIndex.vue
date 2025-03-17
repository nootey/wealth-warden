<script setup lang="ts">
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {computed, onMounted, provide, ref} from "vue";
import LoadingSpinner from "../components/ui/LoadingSpinner.vue";
import {useToastStore} from "../../services/stores/toastStore.ts";
import dateHelper from "../../utils/dateHelper.ts"
import ValidationError from "../components/validation/ValidationError.vue";

import InflowCategories from "../features/inflows/InflowCategories.vue";
import vueHelper from "../../utils/vueHelper.ts";
import type {InflowGroup} from '../../models/inflows.ts';
import type {Statistics} from "../../models/shared.ts";
import BasicStatDisplay from "../components/shared/BasicStatDisplay.vue";
import DisplayMonthlyDate from "../components/shared/DisplayMonthlyDate.vue";
import InflowCreate from "../features/inflows/InflowCreate.vue";
import ReoccurringActionsDisplay from "../components/shared/ReoccurringActionsDisplay.vue";
import {useActionStore} from "../../services/stores/reoccurringActionStore.ts";
import DynamicCategories from "../features/inflows/DynamicCategories.vue";
import YearPicker from "../components/shared/YearPicker.vue";

const inflowStore = useInflowStore();
const toastStore = useToastStore();
const actionStore = useActionStore();

const loadingInflows = ref(true);
const loadingGroupedInflows = ref(true);
const inflows = ref([]);
const groupedInflows = ref<InflowGroup[]>([]);

const addInflowModal = ref(false);
const addCategoryModal = ref(false);
const addDynamicCategoryModal = ref(false);
const inflowStatistics = ref<Statistics[]>([]);

const dataCount = computed(() => {return inflows.value.length});

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

const inflowColumns = ref([
  { field: 'inflow_category', header: 'Category' },
  { field: 'amount', header: 'Amount' },
  { field: 'inflow_date', header: 'Date' },
  { field: 'description', header: 'Description' }
]);

const inflowCategories = computed(() => inflowStore.inflowCategories);
const filteredInflowCategories = ref([]);

onMounted(async () => {
    await init();
});

async function initData() {
  await getData();
  await getGroupedData();
  await inflowStore.getInflowYears();
}

async function init() {
  await initData();
  await inflowStore.getInflowCategories();
  await actionStore.getAllActionsForCategory("inflow");
  sort.value = vueHelper.initSort();
}

async function getData(new_page = null) {

  loadingInflows.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await inflowStore.getInflowsPaginated(
        { ...params.value, year: inflowStore.currentYear },
        page.value
    );
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
  if(dataCount.value < 1)
    return;

  try {

    let response = await inflowStore.getAllGroupedInflows(inflowStore.currentYear);
    groupedInflows.value = response.data;
    loadingGroupedInflows.value = false;
    vueHelper.calculateGroupedStatistics(
        groupedInflows.value,
        inflowStatistics,
        item => item.category_id,
        item => item.category_name,
        item => item.total_amount,
        item => item.month,
        null,
        item => (item as any).category_type
    );
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
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
    case 'add-inflow': {
      addInflowModal.value = value;
      break;
    }
    case 'add-category': {
      addCategoryModal.value = value;
      break;
    }
    case 'add-dynamic-category': {
      addDynamicCategoryModal.value = value;
      break;
    }
    default: {
      break;
    }
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

async function onCellEditComplete(event: any) {

  let inflow_date = dateHelper.mergeDateWithCurrentTime(event?.newData?.inflow_date, "Europe/Ljubljana");

  try {

    let response = await inflowStore.updateInflow({
      id: event.data.id,
      inflow_category_id: event?.newData?.inflow_category.id,
      inflow_category: event?.newData?.inflow_category,
      amount: event?.newData?.amount,
      inflow_date: inflow_date,
      description: event?.newData?.description,
    });

    await getData();
    await getGroupedData();

    toastStore.infoResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }

}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'insertRecAction': {
      await actionStore.getAllActionsForCategory("inflow");
      break;
    }
    default: {
      break;
    }
  }
}

async function updateYear(newYear: number) {
  inflowStore.currentYear = newYear;
  await init();
}

provide("initData", initData)

</script>

<template>

  <Dialog v-model:visible="addInflowModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Add inflow">
    <InflowCreate @insertReoccurringActionEvent="handleEmit('insertRecAction')"></InflowCreate>
  </Dialog>
  <Dialog v-model:visible="addCategoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Inflow categories">
    <InflowCategories :restricted="false"></InflowCategories>
  </Dialog>
  <Dialog v-model:visible="addDynamicCategoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Dynamic categories">
    <DynamicCategories :restricted="false"></DynamicCategories>
  </Dialog>

  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1 fap-2 align-items-center">
        <div class="flex flex-column p-1">
          Select year:
        </div>
        <div>
          <YearPicker records="inflows" :year="inflowStore.currentYear"
                          :availableYears="inflowStore.inflowYears"  @update:year="updateYear" />
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
            <label>Inflows</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-check" label="Create" @click="manipulateDialog('add-inflow', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Inflow categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-arrow-up" label="Manage" @click="manipulateDialog('add-category', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Dynamic inflow categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-arrow-up" label="Manage" @click="manipulateDialog('add-dynamic-category', true)"></Button>
        </div>

      </div>

      <div class="flex flex-row p-1">
        <h3>
          Grouped categories
        </h3>
      </div>

      <DisplayMonthlyDate :groupedValues="groupedInflows" :dataCount="dataCount" />

      <div class="flex flex-row p-1 w-full">
        <h3>
          All inflows
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingInflows" :value="inflows" size="small"
                   editMode="cell" @cell-edit-complete="onCellEditComplete">
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

          <Column v-for="col of inflowColumns" :key="col.field" :field="col.field" :header="col.header" style="width: 25%">
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

            <template #editor="{ data, field }">
              <template v-if="field === 'amount'">
                <InputNumber size="small" v-model="data[field]" mode="currency" currency="EUR" locale="de-DE" autofocus fluid />
              </template>
              <template v-else-if="field === 'inflow_date'">
                <DatePicker v-model="data[field]" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                            style="height: 42px;"/>
              </template>
              <template v-else-if="field === 'inflow_category'">
                <AutoComplete size="small" v-model="data[field]" :suggestions="filteredInflowCategories"
                              @complete="searchInflowCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
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
          Inflows
        </h3>
      </div>

      <BasicStatDisplay :basicStats="inflowStatistics" :limit="false" :dataCount="dataCount" />

      <div class="flex flex-row p-1">
        <h3>
          Reoccurring
        </h3>
      </div>

      <ReoccurringActionsDisplay :categoryItems="actionStore.reoccurringActions" categoryName="inflow" />
    </div>
  </div>
</template>

<style scoped>

</style>