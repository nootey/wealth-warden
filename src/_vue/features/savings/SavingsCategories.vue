<script setup lang="ts">
import {useSavingsStore} from "../../../services/stores/savingsStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, ref} from "vue";
import {numeric, required, requiredIf, minValue, maxValue} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import dateHelper from "../../../utils/dateHelper.ts";
import ValidationError from "../../components/validation/ValidationError.vue";
import LoadingSpinner from "../../components/ui/LoadingSpinner.vue";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();
const savingsCategories = computed(() => savingsStore.savingsCategories);
const newSavingsCategory = ref(initSavingsCategory());
const hasInterest = ref(false);

const props = defineProps<{
  restricted: boolean;
}>();
const loading = ref(false);

const savingsTypes = ref(["fixed", "variable"]);
const filteredSavingsTypes = ref([]);

const isInterestAccount = computed(() => {
  return hasInterest.value;
});

const rules = {
  newSavingsCategory: {
    name: {
      required,
      $autoDirty: true
    },
    savings_type: {
      required,
      $autoDirty: true
    },
    goal_value: {
      numeric,
      minValue: minValue(0),
      maxValue: maxValue(10000000),
      $autoDirty: true,
    },
    interest_rate: {
      numeric,
      minValue: minValue(0),
      maxValue: maxValue(100),
      requiredIfRef: requiredIf(isInterestAccount),
      $autoDirty: true,
    },
  }
}

const v$ = useVuelidate(rules, {newSavingsCategory});

function initSavingsCategory():object {
  return {
    name: null,
    savings_type: null,
    goal_value: null,
    interest_rate: null,
  }
}

const searchSavingsTypes = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredSavingsTypes.value = [...savingsTypes.value];
    } else {
      filteredSavingsTypes.value = savingsTypes.value.filter((record) => {
        return record.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

function toggleAccountType(event: any){
  hasInterest.value = event;
}

async function createNewSavingsCategory() {

  v$.value.newSavingsCategory.$touch();
  if (v$.value.newSavingsCategory.$error) return;

  try {
    let response = await savingsStore.createSavingsCategory({
      id: null,
      name: newSavingsCategory.value.name,
      savings_type: newSavingsCategory.value.savings_type,
      goal_value: newSavingsCategory.value.goal_value,
      interest_rate: newSavingsCategory.value.interest_rate,
    });
    newSavingsCategory.value = initSavingsCategory();
    toastStore.successResponseToast(response);
    v$.value.newSavingsCategory.$reset();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function removeSavingsCategory(id: number) {
  try {
    let response = await savingsStore.deleteSavingsCategory(id);
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
    let response = await savingsStore.updateSavingsCategory({
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
          These are your savings categories. Assign as many as you deem necessary. Once assigned to an savings record, a category can not be deleted.
        </span>
    </div>
    <div class="flex flex-row gap-2 align-items-center w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsCategory.name.$errors[0]?.$message">
          <label>Savings category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-clipboard"></i>
          </InputGroupAddon>
          <InputText size="small" v-model="newSavingsCategory.name" placeholder="Input name"/>
        </InputGroup>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsCategory.savings_type.$errors[0]?.$message">
          <label>Savings type</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newSavingsCategory.savings_type" :suggestions="filteredSavingsTypes"
                      placeholder="Select type" dropdown @complete="searchSavingsTypes"></AutoComplete>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newSavingsCategory.goal_value.$errors[0]?.$message">
          <label>Goal value</label>
        </ValidationError>
        <InputNumber size="small" v-model="newSavingsCategory.goal_value" mode="currency" currency="EUR"
                     locale="de-DE" autofocus fluid placeholder="0,00 €" />
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewSavingsCategory" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <span>Interest account?</span>
      <Checkbox :value="hasInterest" @update:modelValue="toggleAccountType" binary />
    </div>

    <div v-if="hasInterest" class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newSavingsCategory.interest_rate.$errors[0]?.$message">
          <label>Interest rate</label>
        </ValidationError>
        <InputNumber size="small" v-model="newSavingsCategory.interest_rate" inputId="integeronly" :min="0" :max="100" autofocus fluid placeholder="0%" />
      </div>
    </div>

    <div class="flex flex-row p-1 w-full">
      <DataTable dataKey="id" :loading="loading" :value="savingsCategories" size="small"
                 editMode="cell" @cell-edit-complete="onCellEditComplete" sortField="created_at" :sortOrder="1"
                 paginator :rows="5" :rowsPerPageOptions="props.restricted ? [5] : [5, 10, 25]">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

        <Column header="Actions">
          <template #body="slotProps">
            <div class="flex flex-row align-items-center gap-2">
              <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                 @click="removeSavingsCategory(slotProps.data?.id)"></i>
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