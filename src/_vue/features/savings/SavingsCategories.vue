<script setup lang="ts">
import {useSavingsStore} from "../../../services/stores/savingsStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, ref} from "vue";
import {numeric, required, requiredIf, minValue, maxValue} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import dateHelper from "../../../utils/dateHelper.ts";
import ValidationError from "../../components/validation/ValidationError.vue";
import LoadingSpinner from "../../components/ui/LoadingSpinner.vue";
import vueHelper from "../../../utils/vueHelper.ts";
import ColumnHeader from "../../components/shared/ColumnHeader.vue";

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
const accountTypes = ref(["normal", "interest"]);
const filteredSavingsTypes = ref([]);
const filteredAccountTypes = ref([]);

const categoryColumns = ref([
  { field: 'name', header: 'Name' },
  { field: 'savings_type', header: 'Savings type' },
  { field: 'goal_value', header: 'Goal' },
  { field: 'goal_progress', header: 'Progress' },
  { field: 'account_type', header: 'Account' },
  { field: 'interest_rate', header: 'Interest rate' },
  { field: 'accrued_interest', header: 'Accrued Interest' },
]);

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

const searchAccountTypes = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredAccountTypes.value = [...accountTypes.value];
    } else {
      filteredAccountTypes.value = accountTypes.value.filter((record) => {
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
      account_type: hasInterest.value ? "interest" : "normal",
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
  if (event.field === "interest_rate" && event?.newData?.account_type !== "interest") {
    return;
  }

  try {
    let response = await savingsStore.updateSavingsCategory({
      id: event.data.id,
      name: event?.newData?.name,
      savings_type: event?.newData?.savings_type,
      goal_value: event?.newData?.goal_value,
      interest_rate: event?.newData?.interest_rate,
      account_type: event?.newData?.account_type,
    });

    await savingsStore.getSavingsCategories();

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
        <ValidationError :isRequired="true" :message="v$.newSavingsCategory.interest_rate.$errors[0]?.$message">
          <label>Interest rate</label>
        </ValidationError>
        <InputNumber size="small" v-model="newSavingsCategory.interest_rate" :min="0"
                     :max="100"
                     :step="0.1"
                     :minFractionDigits="0"
                     :maxFractionDigits="2"
                     mode="decimal"
                     placeholder="0.0" autofocus fluid />
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

        <Column v-for="col of categoryColumns" :key="col.field" :field="col.field" :header="col.header" style="width: 25%">
          <template #body="{ data, field }">
            <template v-if="['goal_value', 'goal_progress'].includes(col.field)">
              {{ vueHelper.displayAsCurrency(data[col.field]) }}
            </template>
            <template v-else-if="['interest_rate', 'accrued_interest'].includes(col.field)">
              <span v-if="data['account_type'] === 'interest'">{{ field === 'accrued_interest' ? vueHelper.displayAsCurrency(data[field]) : vueHelper.displayAsPercentage(data[field]) }}</span>
              <span v-else> {{ "/" }}</span>
            </template>
            <template v-else>
              {{ data[field] }}
            </template>
          </template>

          <template v-if="!['goal_progress', 'accrued_interest'].includes(col.field)" #editor="{ data, field }">
            <template v-if="field === 'goal_value'">
              <InputNumber size="small" v-model="data[field]" mode="currency" currency="EUR" locale="de-DE" autofocus fluid />
            </template>
            <template v-else-if="field === 'account_type'">
              <AutoComplete size="small" v-model="data[field]" :suggestions="filteredAccountTypes"
                            @complete="searchAccountTypes"  placeholder="Select type" dropdown></AutoComplete>
            </template>
            <template v-else-if="field === 'savings_type'">
              <AutoComplete size="small" v-model="data[field]" :suggestions="filteredSavingsTypes"
                            @complete="searchSavingsTypes"  placeholder="Select type" dropdown></AutoComplete>
            </template>
            <template v-else-if="['interest_rate'].includes(field)">
              <InputNumber v-if="data['account_type'] === 'interest'" size="small" v-model="data[field]" :min="0"
                           :max="100"
                           :step="0.1"
                           :minFractionDigits="0"
                           :maxFractionDigits="2"
                           mode="decimal"
                           placeholder="0.0" />
              <span v-else>{{ "/" }}</span>
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