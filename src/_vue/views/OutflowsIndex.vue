<script setup lang="ts">
import {useToastStore} from "../../services/stores/toastStore.ts";
import {useActionStore} from "../../services/stores/reoccurringActionStore.ts";
import {computed, onMounted, provide, ref} from "vue";
import type {OutflowGroup} from "../../models/outflows.ts";
import type {Statistics} from "../../models/shared.ts";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";
import vueHelper from "../../utils/vueHelper.ts";
import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../components/validation/ValidationError.vue";
import ReoccurringActionsDisplay from "../components/shared/ReoccurringActionsDisplay.vue";
import BasicStatDisplay from "../components/shared/BasicStatDisplay.vue";
import LoadingSpinner from "../components/ui/LoadingSpinner.vue";
import DisplayMonthlyDate from "../components/shared/DisplayMonthlyDate.vue";
import OutflowCategories from "../features/outflows/OutflowCategories.vue";
import OutflowCreate from "../features/outflows/OutflowCreate.vue";
import YearPicker from "../components/shared/YearPicker.vue";

const dataCount = computed(() => {return outflows.value.length});

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

const outflowColumns = ref([
  { field: 'outflow_category', header: 'Category' },
  { field: 'amount', header: 'Amount' },
  { field: 'outflow_date', header: 'Date' },
  { field: 'description', header: 'Description' }
]);

const outflowCategories = computed(() => outflowStore.outflowCategories);
const filteredOutflowCategories = ref([]);

onMounted(async () => {
  await init();
});

async function initData() {
  await getData();
  await getGroupedData();
  await outflowStore.getOutflowYears();
}

async function init() {
  await initData();
  await outflowStore.getOutflowCategories();
  await actionStore.getAllActionsForCategory("outflow");
  sort.value = vueHelper.initSort();
}

async function getData(new_page = null) {

  loadingOutflows.value = true;
  if(new_page)
    page.value = new_page;

  try {

    let paginationResponse = await outflowStore.getOutflowsPaginated(
        { ...params.value, year: outflowStore.currentYear },
        page.value
    );
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
  if(dataCount.value < 1)
    return;

  try {

    let response = await outflowStore.getAllGroupedOutflows(outflowStore.currentYear);
    groupedOutflows.value = response.data;
    loadingGroupedOutflows.value = false;
    vueHelper.calculateGroupedStatistics(
        groupedOutflows.value,
        outflowStatistics,
        item => item.category_id,
        item => item.category_name,
        item => item.total_amount,
        item => item.month,
        item => (item as any).spending_limit,
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

async function removeOutflow(id: number) {
  try {
    let response = await outflowStore.deleteOutflow(id);
    toastStore.successResponseToast(response);
    await  initData();
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

const searchOutflowCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredOutflowCategories.value = [...outflowCategories.value];
    } else {
      filteredOutflowCategories.value = outflowCategories.value.filter((outflowCategory) => {
        return outflowCategory.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function onCellEditComplete(event: any) {

  let outflow_date = dateHelper.mergeDateWithCurrentTime(event?.newData?.outflow_date, "Europe/Ljubljana");

  try {

    let response = await outflowStore.updateOutflow({
      id: event.data.id,
      outflow_category_id: event?.newData?.outflow_category.id,
      outflow_category: event?.newData?.outflow_category,
      amount: event?.newData?.amount,
      outflow_date: outflow_date,
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
      await actionStore.getAllActionsForCategory("outflow");
      break;
    }
    default: {
      break;
    }
  }
}

async function updateYear(newYear: number) {
  outflowStore.currentYear = newYear;
  await init();
}

provide("initData", initData)

</script>

<template>

  <Dialog v-model:visible="addOutflowModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Add outflow">
    <OutflowCreate @insertReoccurringActionEvent="handleEmit('insertRecAction')"></OutflowCreate>
  </Dialog>
  <Dialog v-model:visible="addCategoryModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '800px'}" header="Outflow categories">
    <OutflowCategories :restricted="false"></OutflowCategories>
  </Dialog>

  <div class="flex w-full p-2">
    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1 fap-2 align-items-center">
        <div class="flex flex-column p-1">
          Select year:
        </div>
        <div>
          <YearPicker records="outflows" :year="outflowStore.currentYear"
                      :availableYears="outflowStore.outflowYears"  @update:year="updateYear" />
        </div>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Manage entries
        </h3>
      </div>

      <div class="flex flex-row p-1 w-full gap-3">
        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Outflows</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-check" label="Create" @click="manipulateDialog('add-outflow', true)"></Button>
        </div>

        <div class="flex flex-column w-6 justify-content-center align-items-center">
          <ValidationError :isRequired="false" message="">
            <label>Outflow categories</label>
          </ValidationError>
          <Button class="w-6" icon="pi pi-file-arrow-up" label="Manage" @click="manipulateDialog('add-category', true)"></Button>
        </div>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Outflows by month
        </h3>
      </div>


      <DisplayMonthlyDate :groupedValues="groupedOutflows" :dataCount="dataCount"/>

      <div class="flex flex-row p-1 w-full">
        <h3>
          All outflows
        </h3>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingOutflows" :value="outflows" size="small"
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
                   @click="removeOutflow(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column v-for="col of outflowColumns" :key="col.field" :field="col.field" :header="col.header" style="width: 25%">
            <template #body="{ data, field }">
              <template v-if="field === 'amount'">
                {{ vueHelper.displayAsCurrency(data.amount)}}
              </template>
              <template v-else-if="field === 'outflow_date'">
                {{ dateHelper.formatDate(data?.outflow_date, true) }}
              </template>
              <template v-else-if="field === 'outflow_category'">
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
              <template v-else-if="field === 'outflow_date'">
                <DatePicker v-model="data[field]" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                            style="height: 42px;"/>
              </template>
              <template v-else-if="field === 'outflow_category'">
                <AutoComplete size="small" v-model="data[field]" :suggestions="filteredOutflowCategories"
                              @complete="searchOutflowCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
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
        <h1>
          Statistics
        </h1>
      </div>

      <div class="flex flex-row p-1">
        <h3>
          Outflows
        </h3>
      </div>

      <BasicStatDisplay :basicStats="outflowStatistics" :limit="true" :dataCount="dataCount"/>

      <div class="flex flex-row p-1">
        <h3>
          Reoccurring
        </h3>
      </div>

      <ReoccurringActionsDisplay categoryName="outflow" :categoryItems="actionStore.reoccurringActions" />
    </div>
  </div>
</template>

<style scoped>

</style>