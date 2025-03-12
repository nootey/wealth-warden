<script setup lang="ts">

import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../Validation/ValidationError.vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {computed, ref} from "vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const props = defineProps<{
  restricted: boolean;
}>();

const inflowCategories = computed(() => inflowStore.inflowCategories);
const newInflowCategory = ref(initInflowCategory());
const loading = ref(false);

const inflowCategoryRules = {
  newInflowCategory: {
    name: {
      required,
      $autoDirty: true
    }
  }
}

const v$ = useVuelidate(inflowCategoryRules, {newInflowCategory: newInflowCategory});

function initInflowCategory():object {
  return {
    name: null,
  }
}

async function createNewInflowCategory() {

  v$.value.newInflowCategory.$touch();
  if (v$.value.newInflowCategory.$error) return;

  try {
    let response = await inflowStore.createInflowCategory({name: newInflowCategory.value.name});
    newInflowCategory.value = initInflowCategory();
    toastStore.successResponseToast(response);
    v$.value.newInflowCategory.$reset();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function removeInflowCategory(id: number) {
  try {
    let response = await inflowStore.deleteInflowCategory(id);
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onCellEditComplete(event: any) {
  if(props.restricted) {
    return;
  }
  try {
    let response = await inflowStore.updateInflowCategory({
      id: event.data.id,
      name: event?.newData?.name,
    });

    toastStore.infoResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }

}

</script>

<template>
  <div class="flex flex-column w-full p-1 gap-4">
    <div v-if="!restricted" class="flex flex-row p-1 w-full">
        <span>
          These are your inflow categories. Assign as many as you deem necessary. Once assigned to an inflow record, a category can not be deleted.
        </span>
    </div>
    <div class="flex flex-row gap-2 align-items-center w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflowCategory.name.$errors[0]?.$message">
          <label>Inflow category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-clipboard"></i>
          </InputGroupAddon>
          <InputText v-model="newInflowCategory.name"/>
        </InputGroup>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewInflowCategory" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row p-1 w-full">
      <DataTable dataKey="id" :loading="loading" :value="inflowCategories" size="small"
                 editMode="cell" @cell-edit-complete="onCellEditComplete" sortField="created_at" :sortOrder="1"
                 paginator :rows="5" :rowsPerPageOptions="props.restricted ? [5] : [5, 10, 25]">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

        <Column header="Actions">
          <template #body="slotProps">
            <div class="flex flex-row align-items-center gap-2">
              <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                 @click="removeInflowCategory(slotProps.data?.id)"></i>
            </div>
          </template>
        </Column>

        <Column field="name" header="Name" :sortable="true">
          <template v-if="!props.restricted" #editor="{ data, field }">
            <InputText  size="small" v-model="data[field]" autofocus fluid />
          </template>
        </Column>
        <Column field="created_at" header="Created" :sortable="true">
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