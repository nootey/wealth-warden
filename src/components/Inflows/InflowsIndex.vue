<script setup lang="ts">
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {ref} from "vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {useToastStore} from "../../services/stores/toastStore.ts";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const loadingInflows = ref(false);

const inflowCategories = ref([]);
const filteredInflowCategories = ref([]);
const inflows = ref([]);
const newInflowCategory = ref(null);
const newInflow = ref(initInflow());

init()

async function init() {
  await getInflowsPaginated();
  await getInflowCategories();
}

function initInflow():object {
  return {
    amount: 0.00,
    inflowCategory: [],
    date: new Date().toISOString(),
  }
}

async function getInflowsPaginated() {
  try {
    let paginationResponse = await inflowStore.getInflowsPaginated();
    inflows.value = paginationResponse.data;
  } catch (error) {
    console.error('Error during login:', error);
  }
}

async function getInflowCategories() {
  try {
    inflowCategories.value = await inflowStore.getInflowCategories();
  } catch (error) {
    console.error('Error during login:', error);
  }
}

async function createNewInflow() {
  try {
    let date = new Date(newInflow.value.date).toISOString()
    let response = await inflowStore.createInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: date});
    toastStore.successResponseToast(response);
    await getInflowsPaginated();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function createNewInflowCategory() {
  try {
    let response = await inflowStore.createInflowCategory({name: newInflowCategory.value});
    toastStore.successResponseToast(response);
    await getInflowCategories();
  } catch (error) {
    toastStore.errorResponseToast(error);
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

async function editInflow(id: number) {
  console.log(id)
}

async function removeInflow(id: number) {
  try {
    let response = await inflowStore.deleteInflow(id);
    toastStore.successResponseToast(response);
    await getInflowsPaginated();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function editInflowCategory(id: number) {
  console.log(id)
}

async function removeInflowCategory(id: number) {
  try {
    let response = await inflowStore.deleteInflowCategory(id);
    toastStore.successResponseToast(response);
    await getInflowCategories();
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
          <label>Category</label>
          <InputGroup>
            <InputGroupAddon>
              <i class="pi pi-user"></i>
            </InputGroupAddon>
            <AutoComplete size="small" v-model="newInflow.inflowCategory" :suggestions="filteredInflowCategories"
                          @complete="searchInflowCategory" option-label="name" dropdown></AutoComplete>
          </InputGroup>
        </div>

        <div class="flex flex-column">
          <label>Amount</label>
          <InputGroup>
            <InputGroupAddon>
              <i class="pi pi-book"></i>
            </InputGroupAddon>
            <InputNumber size="small" v-model="newInflow.amount"></InputNumber>
          </InputGroup>
        </div>

        <div class="flex flex-column">
          <label>Date</label>
          <DatePicker size="small" v-model="newInflow.date"/>
        </div>

        <div class="flex flex-column">
          <label>Submit</label>
          <Button icon="pi pi-cart-plus" @click="createNewInflow" />
        </div>

      </div>

      <div class="flex flex-row gap-2" style="border-top: 1px solid var(--text-primary)">
        <DataTable dataKey="id" :loading="loadingInflows" :value="inflows" class="p-datatable-sm">
          <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
          <template #loading> <LoadingSpinner></LoadingSpinner> </template>

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
          <Column field="inflow_date" header="Date"></Column>

        </DataTable>
      </div>

    </div>

    <div class="flex flex-column w-3 p-3 gap-3" style="border-left: 1px solid var(--text-primary);">
      <div class="flex flex-row p-1">
        <h1>
          Inflow Categories
        </h1>
      </div>
      <div class="flex flex-row p-1">
        <span>
          These are your inflow categories. Assign as many as you deem necessary. Once assigned to an inflow record, a category can not be deleted.
        </span>
      </div>
      <div class="flex flex-row p-1">
          <FloatLabel variant="in">
            <InputText id="in_label" v-model="newInflowCategory" variant="filled" @keydown.enter="createNewInflowCategory" />
            <label for="in_label">New inflow category</label>
          </FloatLabel>

      </div>
      <div class="flex flex-row p-1">
        <DataTable dataKey="id" :loading="loadingInflows" :value="inflowCategories" class="p-datatable-sm">
          <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
          <template #loading> <LoadingSpinner></LoadingSpinner> </template>

          <Column header="Actions">
            <template #body="slotProps">
              <div class="flex flex-row align-items-center gap-2">
                <i class="pi pi-pencil hover_icon"
                   @click="editInflowCategory(slotProps.data?.id)"></i>
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeInflowCategory(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>

          <Column field="name" header="Name"></Column>
          <Column field="created_at" header="Created"></Column>
        </DataTable>
      </div>

    </div>


  </div>
</template>

<style scoped>

</style>