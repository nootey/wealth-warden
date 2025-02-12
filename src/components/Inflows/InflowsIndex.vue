<script setup lang="ts">
import {useGeneralStore} from "../../services/stores/general.ts";
import {ref} from "vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";

const generalStore = useGeneralStore();

const loadingInflows = ref(false);

const inflowTypes = ref([]);
const filteredInflowTypes = ref([]);
const inflows = ref([]);
const newInflowType = ref(null);
const newInflow = ref(initInflow());

init()

async function init() {
  await getInflowsPaginated();
  await getInflowTypes();
}

function initInflow():object {
  return {
    amount: 0.00,
    inflowType: [],
    date: new Date().toISOString(),
  }
}

async function getInflowsPaginated() {
  try {
    let paginationResponse = await generalStore.getInflowsPaginated();
    inflows.value = paginationResponse.data;
    console.log(paginationResponse);
    console.log(inflows.value)
  } catch (error) {
    console.error('Error during login:', error);
  }
}

async function getInflowTypes() {
  try {
    inflowTypes.value = await generalStore.getInflowTypes();
  } catch (error) {
    console.error('Error during login:', error);
  }
}

async function createNewInflow() {
  try {
    let date = new Date(newInflow.value.date).toISOString()
    await generalStore.createInflow({
      inflow_type_id: newInflow.value.inflowType.id,
      inflow_type: newInflow.value.inflowType,
      amount: newInflow.value.amount,
      inflow_date: date});
  } catch (error) {
    console.error('Error during login:', error);
  }
}

async function createNewInflowType() {
  console.log("test")
  try {
    inflowTypes.value = await generalStore.createInflowType({name: newInflowType.value});
  } catch (error) {
    console.error('Error during login:', error);
  }
}

const searchInflowType = (event) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredInflowTypes.value = [...inflowTypes.value];
    } else {
      filteredInflowTypes.value = inflowTypes.value.filter((inflowType) => {
        return inflowType.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function editInflow(id) {
  console.log(id)
}

async function removeInflow(id) {
  console.log(id)
}

async function editInflowType(id) {
  console.log(id)
}

async function removeInflowType(id) {
  console.log(id)
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
          <label>Type</label>
          <InputGroup>
            <InputGroupAddon>
              <i class="pi pi-user"></i>
            </InputGroupAddon>
            <AutoComplete size="small" v-model="newInflow.inflowType" :suggestions="filteredInflowTypes"
                          @complete="searchInflowType" option-label="name" dropdown></AutoComplete>
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

          <Column field="id" header="ID"></Column>
          <Column field="amount" header="Amount"></Column>
          <Column field="inflow_date" header="Date"></Column>
          <Column field="inflow_category" header="Category"></Column>
          <Column field="inflow_date" header="Date"></Column>
        </DataTable>
      </div>

    </div>

    <div class="flex flex-column w-3 p-3 gap-3" style="border-left: 1px solid var(--text-primary);">
      <div class="flex flex-row p-1">
        <h1>
          Inflow types
        </h1>
      </div>
      <div class="flex flex-row p-1">
        <span>
          These are your inflow types. Think of them as spending categories.
        </span>
      </div>
      <div class="flex flex-row p-1">
          <FloatLabel variant="in">
            <InputText id="in_label" v-model="newInflowType" variant="filled" @keydown.enter="createNewInflowType" />
            <label for="in_label">New inflow type</label>
          </FloatLabel>

      </div>
      <div class="flex flex-row p-1">
        <DataTable dataKey="id" :loading="loadingInflows" :value="inflowTypes" class="p-datatable-sm">
          <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
          <template #loading> <LoadingSpinner></LoadingSpinner> </template>

          <Column header="Actions">
            <template #body="slotProps">
              <div class="flex flex-row align-items-center gap-2">
                <i class="pi pi-pencil hover_icon"
                   @click="editInflowType(slotProps.data?.id)"></i>
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeInflowType(slotProps.data?.id)"></i>
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