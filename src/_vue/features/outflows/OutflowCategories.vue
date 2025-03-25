<script setup lang="ts">

import dateHelper from "../../../utils/dateHelper.ts";
import ValidationError from "../../components/validation/ValidationError.vue";
import LoadingSpinner from "../../components/ui/LoadingSpinner.vue";
import {computed, ref} from "vue";
import {integer, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useOutflowStore} from "../../../services/stores/outflowStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import vueHelper from "../../../utils/vueHelper.ts";

const outflowStore = useOutflowStore();
const toastStore = useToastStore();

const props = defineProps<{
  restricted: boolean;
}>();


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

const categoryColumns = ref([
  { field: 'name', header: 'Name' },
  { field: 'outflow_type', header: 'Outflow type' },
  { field: 'spending_limit', header: 'Spending limit' },
]);

const outflowTypes = ref(["fixed", "variable"]);
const filteredOutflowTypes = ref([]);

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
  if(props.restricted) {
    return;
  }

  try {
    let response = await outflowStore.updateOutflowCategory({
      id: event.data.id,
      name: event?.newData?.name,
      spending_limit: event?.newData?.spending_limit,
      outflow_type: event?.newData?.outflow_type,
    });

    toastStore.infoResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }

}

const searchOutflowType = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredOutflowTypes.value = [...outflowTypes.value];
    } else {
      filteredOutflowTypes.value = outflowTypes.value.filter((record) => {
        return record.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

</script>

<template>
  <div class="flex flex-column w-full p-1 gap-4">
    <div v-if="!restricted" class="flex flex-row p-1 w-full">
        <span>
          These are your outflow categories. Assign as many as you deem necessary.
          Once assigned to an outflow record, a category can not be deleted.
        </span>
    </div>
    <div v-if="!restricted" class="flex flex-row p-1 w-full">
        <span>
          Define spending limits. These will serve as thresholds for each category, which you shouldn't cross.
        </span>
    </div>
    <div v-if="!restricted" class="flex flex-row p-1 w-full">
        <span>
          Outflow type will define if the category should be budgeted automatically or not.
        </span>
    </div>

    <div class="flex flex-row gap-2 align-items-center w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflowCategory.name.$errors[0]?.$message">
          <label>Name</label>
        </ValidationError>
        <InputText size="small" v-model="newOutflowCategory.name" placeholder="Input name"/>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflowCategory.outflow_type.$errors[0]?.$message">
          <label>Outflow type</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newOutflowCategory.outflow_type" :suggestions="filteredOutflowTypes"
                      placeholder="Select type" dropdown @complete="searchOutflowType"></AutoComplete>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newOutflowCategory.spending_limit.$errors[0]?.$message">
          <label>Spending limit</label>
        </ValidationError>
        <InputNumber size="small" v-model="newOutflowCategory.spending_limit" mode="currency"
                     currency="EUR" locale="de-DE" autofocus fluid placeholder="0,00 €"/>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewOutflowCategory" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row p-1 w-full">
      <DataTable class="w-full" dataKey="id" :loading="loading" :value="outflowCategories" size="small"
                 editMode="cell" @cell-edit-complete="onCellEditComplete" sortField="outflow_type" :sortOrder="1"
                 paginator :rows="5" :rowsPerPageOptions="props.restricted ? [5] : [5, 10, 25]">
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

        <Column v-for="col of categoryColumns" :key="col.field" :field="col.field" :header="col.header"
                style="width: 33%" :sortable="true" >
          <template #body="{ data, field }">
            <template v-if="field === 'spending_limit'">
              {{ vueHelper.displayAsCurrency(data.spending_limit)}}
            </template>
            <template v-else-if="field === 'created_at'">
              {{ dateHelper.formatDate(data.created_at, true) }}
            </template>
            <template v-else>
              {{ data[field] }}
            </template>
          </template>

          <template v-if="!props.restricted" #editor="{ data, field }">
            <template v-if="field === 'spending_limit'">
              <InputNumber size="small" v-model="data[field]" mode="currency" currency="EUR" locale="de-DE" autofocus fluid />
            </template>
            <template v-else-if="field === 'created_at'">
              <DatePicker v-model="data[field]" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                          style="height: 42px;"/>
            </template>
            <template v-else-if="field === 'outflow_type'">
              <AutoComplete size="small" v-model="data[field]" :suggestions="filteredOutflowTypes"
                            placeholder="Select type" dropdown @complete="searchOutflowType"></AutoComplete>
            </template>
            <template v-else>
              <InputText size="small" v-model="data[field]" autofocus fluid />
            </template>
          </template>
        </Column>

      </DataTable>
    </div>

  </div>
</template>

<style scoped>

</style>