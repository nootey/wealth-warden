<script setup lang="ts">

import dateHelper from "../../utils/dateHelper.ts";
import ValidationError from "../Validation/ValidationError.vue";
import LoadingSpinner from "../Utils/LoadingSpinner.vue";
import {computed, onMounted, ref} from "vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";

const inflowStore = useInflowStore();
const outflowStore = useOutflowStore();
const toastStore = useToastStore();

const newDynamicCategory = ref(initDynamicCategory());
const loading = ref(false);

const categoryTypes = ref([
  {name: "inflow"},
  {name: "outflow"},
]);
const dynamicCategories = computed(() => inflowStore.dynamicCategories);
const inflowCategories = computed(() => inflowStore.inflowCategories);
const outflowCategories = computed(() => outflowStore.outflowCategories);
const mergedCategories = computed(() => {
  return [
    ...(dynamicCategories.value || []).map(category => ({
      ...category,
      category_type: 'dynamic'
    })),
    ...(inflowCategories.value || []).map(category => ({
      ...category,
      category_type: 'inflow'
    }))
  ];
});
const selectedCategories = ref([]);

const filteredCategoryTypes = ref([]);

onMounted(async () => {
  await outflowStore.getOutflowCategories();
  await inflowStore.getDynamicCategories();
})

const inflowCategoryRules = {
  newDynamicCategory: {
    name: {
      required,
      $autoDirty: true
    },
    primary_links: {
      required,
      $autoDirty: true
    },
    secondary_links: {
      $autoDirty: true
    }
  }
}

const v$ = useVuelidate(inflowCategoryRules, {newDynamicCategory: newDynamicCategory});

function initDynamicCategory():object {
  return {
    name: null,
    primary_type: "inflow",
    primary_links: [],
    secondary_type: "outflow",
    secondary_links: [],
  }
}

const searchCategoryType = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredCategoryTypes.value = [...categoryTypes.value];
    } else {
      filteredCategoryTypes.value = categoryTypes.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function updateCategoryType(){
  switch(newDynamicCategory.value?.type?.name) {
    case "inflow": {
      selectedCategories.value = [...inflowCategories.value];
      break;
    }
    case "outflow": {
      selectedCategories.value = [...outflowCategories.value];
      break;
    }
    default: {
      selectedCategories.value = [];
      break;
    }
  }
}

async function createNewDynamicCategory() {

  v$.value.newDynamicCategory.$touch();
  if (v$.value.newDynamicCategory.$error) return;

  try {
    let response = await inflowStore.createDynamicCategory({
      id: null,
      name: newDynamicCategory.value.name,
    },
{
      primary_links: newDynamicCategory.value.primary_links,
      primary_type: newDynamicCategory.value.primary_type,
      secondary_links: newDynamicCategory.value.secondary_links,
      secondary_type: newDynamicCategory.value.secondary_type,
    }
    );
    newDynamicCategory.value = initDynamicCategory();
    await inflowStore.getDynamicCategories();
    toastStore.successResponseToast(response);
    v$.value.newDynamicCategory.$reset();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function removeDynamicCategory(id: number) {
  console.log(id)
  return;
  try {
    let response = await inflowStore.deleteInflowCategory(id);
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onCellEditComplete(event: any) {
  console.log(event);
  return;
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

    <div class="flex flex-row p-1 w-full">
        <span>
          These are your custom categories. Each entry represents a difference between the linked categories.
        </span>
    </div>
    <div class="flex flex-row p-1 w-full">
        <span>
          For example,
          if you assign your "Salary" inflow category as the primary link, and your work-related fixed expenses as the secondary link,
          the category will represent a "True salary" entry. You can also use these created categories, to create new custom categories.
        </span>
    </div>

    <div class="flex flex-row gap-2 align-items-center w-full">

      <div class="flex flex-column w-5">
        <ValidationError :isRequired="true" :message="v$.newDynamicCategory.name.$errors[0]?.$message">
          <label>Name</label>
        </ValidationError>
        <InputText size="small" v-model="newDynamicCategory.name"/>
      </div>

      <div class="flex flex-column w-4">
        <ValidationError :isRequired="true" :message="v$.newDynamicCategory.primary_links.$errors[0]?.$message">
          <label>Primary link</label>
        </ValidationError>
        <MultiSelect v-model="newDynamicCategory.primary_links" :options="mergedCategories" optionLabel="name" filter
                     placeholder="Select category" size="small"></MultiSelect>
      </div>

      <div class="flex flex-column w-4">
        <ValidationError :isRequired="false" :message="v$.newDynamicCategory.secondary_links.$errors[0]?.$message">
          <label>Secondary link</label>
        </ValidationError>
        <MultiSelect v-model="newDynamicCategory.secondary_links"
                     :options="outflowCategories" optionLabel="name" filter
        placeholder="Select category" size="small"></MultiSelect>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewDynamicCategory" style="height: 42px;" />
      </div>
    </div>

    <hr>

    <div class="flex flex-row p-1 w-full">
      <DataTable dataKey="id" class="w-full" :loading="loading" :value="dynamicCategories" size="small"
                 editMode="cell" @cell-edit-complete="onCellEditComplete" sortField="created_at" :sortOrder="1"
                 paginator :rows="5" :rowsPerPageOptions="[5, 10, 25]">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

        <Column header="Actions">
          <template #body="slotProps">
            <div class="flex flex-row align-items-center gap-2">
              <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                 @click="removeDynamicCategory(slotProps.data?.id)"></i>
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