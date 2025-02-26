<script setup lang="ts">

import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../Validation/ValidationError.vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {computed, ref} from "vue";
import {integer, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";

const outflowStore = useOutflowStore();
const toastStore = useToastStore();

const outflowCategories = computed(() => outflowStore.outflowCategories);
const newOutflowCategory = ref(initOutflowCategory());
const loading = ref(false);

const outflowCategoryRules = {
  newOutflowCategory: {
    name: {
      required,
      $autoDirty: true
    },
    spending_limit: {
      required,
      integer,
      $autoDirty: true,
    },
    outflow_type: {
      required,
      $autoDirty: true,
    }
  }
}

const v$ = useVuelidate(outflowCategoryRules, {newOutflowCategory: newOutflowCategory});

function initOutflowCategory():object {
  return {
    name: null,
    spending_limit: 0,
    outflow_type: null,
  }
}

async function createNewOutflowCategory() {

  v$.value.newOutflowCategory.$touch();
  if (v$.value.newOutflowCategory.$error) return;

  try {
    let response = await outflowStore.createOutflowCategory({
      id: null,
      name: newOutflowCategory.value.name,
      spending_limit: newOutflowCategory.value.spending_limit,
      outflow_type: newOutflowCategory.value.outflow_type
    });
    newOutflowCategory.value = initOutflowCategory();
    toastStore.successResponseToast(response);
    v$.value.newOutflowCategory.$reset();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function removeOutflowCategory(id: number) {
  try {
    let response = await outflowStore.deleteOutflowCategory(id);
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onCellEditComplete(event: any) {

  try {
    let response = await outflowStore.updateOutflowCategory({
      id: event.data.id,
      name: event?.newData?.name,
      spending_limit: event?.newData?.spending_limit,
      outflow_type: event?.newData?.outflow_type,
    });

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }

}

</script>

<template>
  <div class="flex flex-column w-full p-1 gap-4">
    <div class="flex flex-row p-1 w-full">
      <h2>
        Outflow Categories
      </h2>
    </div>
    <div class="flex flex-row p-1 w-full">
        <span>
          These are your outflow categories. Assign as many as you deem necessary. Once assigned to an outflow record, a category can not be deleted.
        </span>
    </div>
    <div class="flex flex-row gap-2 align-items-center w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflowCategory.name.$errors[0]?.$message">
          <label>Outflow category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-clipboard"></i>
          </InputGroupAddon>
          <InputText v-model="newOutflowCategory.name"/>
        </InputGroup>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewOutflowCategory" style="height: 42px;" />
      </div>
    </div>

    {{ outflowCategories }}

    <div class="flex flex-row p-1 w-full">
      <DataTable dataKey="id" :loading="loading" :value="outflowCategories" size="small"
                 editMode="cell" @cell-edit-complete="onCellEditComplete">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

        <Column header="Actions">
          <template #body="slotProps">
            <div class="flex flex-row align-items-center gap-2">
              <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                 @click="removeOutflowCategory(slotProps.data?.id)"></i>
            </div>
          </template>
        </Column>

        <Column field="name" header="Name">
          <template #editor="{ data, field }">
            <InputText size="small" v-model="data[field]" autofocus fluid />
          </template>
        </Column>
        <Column field="created_at" header="Created">
          <template #body="slotProps">
            {{ dateHelper.formatDate(slotProps.data?.created_at, true) }}
          </template>
        </Column>
      </DataTable>
    </div>

  </div>
</template>

<style scoped>

</style>