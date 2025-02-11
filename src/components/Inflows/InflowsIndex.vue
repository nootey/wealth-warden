<script setup lang="ts">
import {useGeneralStore} from "../../services/stores/general.ts";
import {ref} from "vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";

const generalStore = useGeneralStore();

const loadingInflows = ref(false);

const inflowTypes = ref([]);
const inflows = ref([]);
const newInflowType = ref(null);

init()

async function init() {

  try {
    inflowTypes.value = await generalStore.getInflowTypes();
    console.log(inflowTypes.value)
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

async function editInflow(id) {
  console.log(id)
}

async function removeInflow(id) {
  console.log(id)
}
</script>

<template>
  <div class="flex w-12 p-2 m-1">
    <div class="flex w-9">
      <DataTable dataKey="id" :loading="loadingInflows" :value="inflowTypes" class="p-datatable-sm">
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
      </DataTable>

    </div>
    <div class="flex w-3" style="background: blue;">
      thin
    </div>
<!--    <h1>Inflows</h1>-->
<!--    <p> Define your inflows. You can define reoccurring entries, and dynamically add random ones after.</p>-->

<!--    <FloatLabel variant="in">-->
<!--      <InputText id="in_label" v-model="newInflowType" variant="filled" @keydown.enter="createNewInflowType" />-->
<!--      <label for="in_label">In Label</label>-->
<!--    </FloatLabel>-->

  </div>
</template>

<style scoped>

</style>