<script setup lang="ts">
import {useSavingsStore} from "../../../services/stores/savingsStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, ref, watch} from "vue";
import {numeric, required, requiredIf, minValue, maxValue, helpers, integer} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import dateHelper from "../../../utils/dateHelper.ts";
import ValidationError from "../../components/validation/ValidationError.vue";
import LoadingSpinner from "../../components/ui/LoadingSpinner.vue";
import vueHelper from "../../../utils/vueHelper.ts";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();
const savingsCategories = computed(() => savingsStore.savingsCategories);
const newSavingsCategory = ref(initSavingsCategory());
const newReoccurringRecord = ref(initSavingsCategory(true));
const hasInterest = ref(false);
const isReoccurring = ref(false);
const newAllocation = ref(initAllocation());

const props = defineProps<{
  restricted: boolean;
  availableAllocation: any;
}>();

const emit = defineEmits<{
  (event: 'insertReoccurringActionEvent'): void;
}>();

const loading = ref(false);

const savingsTypes = ref(["fixed", "variable"]);
const accountTypes = ref(["normal", "interest"]);
const filteredSavingsTypes = ref([]);
const filteredAccountTypes = ref([]);

const reoccurrenceUnits = ref([
  {name: "Days"},
  {name: "Weeks"},
  {name: "Months"},
  {name: "Year"},
])
const filteredReoccurrenceUnits = ref([]);

const categoryColumns = ref([
  { field: 'name', header: 'Name' },
  { field: 'savings_type', header: 'Savings type' },
  { field: 'goal_target', header: 'Goal' },
  { field: 'account_type', header: 'Account' },
  { field: 'interest_rate', header: 'Interest rate' },
  { field: 'accrued_interest', header: 'Accrued Interest' },
]);

const isInterestAccount = computed(() => {
  return hasInterest.value;
});

const isEndDateValid = (value: string | null) => {
  if (!value) return true; // Allow null values
  return new Date(value) > new Date(newReoccurringRecord.value?.startDate);
};

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
    goal_target: {
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
  },
  newReoccurringRecord: {
    startDate: {
      required,
      $autoDirty: true
    },
    endDate: {
      $autoDirty: true,
      isEndDateValid: helpers.withMessage('End date must be higher than starting date.', isEndDateValid),
    },
    intervalValue: {
      required,
      integer,
      minValue: minValue(0),
      maxValue: maxValue(9),
      $autoDirty: true
    },
    intervalUnit: {
      required,
      $autoDirty: true
    },
  },
  newAllocation: {
    method:{
      $autoDirty: true
    },
    allocation: {
      numeric,
      minValue: 0,
      maxValue: 1000000000,
      $autoDirty: true
    },
    allocated_value:{
      required,
      $autoDirty: true
    },
  }
}

const v$ = useVuelidate(rules, {newSavingsCategory, newReoccurringRecord, newAllocation});

watch(
    () => [newAllocation.value.method, newAllocation.value.allocation],
    ([method, allocation]) => {
      if (allocation == null) {

        newAllocation.value.allocated_value = 0;
        return;
      }

      if (method === 'absolute') {
        newAllocation.value.allocated_value = allocation;
      } else if (method === 'percentage') {

        if(newAllocation.value.allocation > 100)
          newAllocation.value.allocation = 100;
        newAllocation.value.allocated_value = (allocation/100) * (props.availableAllocation?.allocated_value ?? 0);
      }
    },
    { immediate: true } // runs initially too
);

function initSavingsCategory(isReoccurring: boolean = false):object {

  if (isReoccurring) {
    return {
      startDate: dateHelper.formatDate(new Date(), true),
      endDate: null,
      intervalValue: 1,
      intervalUnit: {name: "Months"},
      description: null,
    };
  }

  return {
    name: null,
    savings_type: null,
    goal_target: null,
    interest_rate: null,
    method: "percentage",
    allocation: null,
    allocated_value: null,
  }
}

function initAllocation(){
  return {
    method: "percentage",
    allocation: 0,
    allocated_value: 0,
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

function toggleReoccurrence(event: any){
  isReoccurring.value = event;
}

async function validateForm(){
  let isValidReoccurring = true;
  const isValidCategory = await v$.value.newSavingsCategory.$validate();

  if (newSavingsCategory.value.savings_type === "fixed") {
    isValidReoccurring = await v$.value.newReoccurringRecord.$validate();
  }

  if (!isValidReoccurring) return true;
  return !isValidCategory;

}

async function createNewSavingsCategory() {

  if (await validateForm()) return;

  try {

    let start_date = dateHelper.mergeDateWithCurrentTime(newReoccurringRecord.value.start_date, "Europe/Ljubljana");
    let end_date = newReoccurringRecord.value.end_date ? dateHelper.mergeDateWithCurrentTime(newReoccurringRecord.value.end_date, "Europe/Ljubljana") : null;

    let reoccurring_action = {
      category_type: "savings_category",
      start_date: start_date,
      end_date: end_date,
      interval_unit: newReoccurringRecord.value.intervalUnit.name,
      interval_value: newReoccurringRecord.value.intervalValue
    }

    let response = await savingsStore.createSavingsCategory({
      id: null,
      name: newSavingsCategory.value.name,
      savings_type: newSavingsCategory.value.savings_type,
      goal_target: newSavingsCategory.value.goal_target,
      interest_rate: newSavingsCategory.value.interest_rate,
      account_type: hasInterest.value ? "interest" : "normal",
    },
        newSavingsCategory.value.savings_type === "fixed" || isReoccurring.value,
        reoccurring_action,
        newAllocation.value.allocated_value ?? 0
    )
    newSavingsCategory.value = initSavingsCategory();
    emit("insertReoccurringActionEvent");
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
      goal_target: event?.newData?.goal_target,
      interest_rate: event?.newData?.interest_rate,
      account_type: event?.newData?.account_type,
    });

    await savingsStore.getSavingsCategories();

    toastStore.infoResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }

}

const searchReoccurrenceUnit = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredReoccurrenceUnits.value = [...reoccurrenceUnits.value];
    } else {
      filteredReoccurrenceUnits.value = reoccurrenceUnits.value.filter((item) => {
        return item.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
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
        <ValidationError :isRequired="false" :message="v$.newSavingsCategory.goal_target.$errors[0]?.$message">
          <label>Goal value</label>
        </ValidationError>
        <InputNumber size="small" v-model="newSavingsCategory.goal_target" mode="currency" currency="EUR"
                     locale="de-DE" autofocus fluid placeholder="0,00 €" />
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewSavingsCategory" style="height: 42px;" />
      </div>
    </div>

    <div v-if="newSavingsCategory.savings_type === 'fixed' || isReoccurring" class="flex flex-row w-full gap-2">
      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="null">
          <label>Value</label>
        </ValidationError>
        <SelectButton v-model="newAllocation.method" size="small" :options="['absolute', 'percentage']" />
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newAllocation.allocation.$errors[0]?.$message">
          <label>Allocation</label>
        </ValidationError>
        <InputNumber v-if="newAllocation.method === 'absolute'" size="small" v-model="newAllocation.allocation" mode="currency" currency="EUR"
                     locale="de-DE" placeholder="0,00 €"></InputNumber>
        <InputNumber v-if="newAllocation.method === 'percentage'" size="small" v-model="newAllocation.allocation" :min="0"
                     :max="100"
                     :step="0.1"
                     :minFractionDigits="0"
                     :maxFractionDigits="2"
                     mode="decimal"
                     placeholder="0.0" autofocus fluid></InputNumber>
      </div>

      <div v-if="newAllocation.allocated_value > 0" class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newAllocation.allocation.$errors[0]?.$message">
          <label>Allocated value</label>
        </ValidationError>
        <InputNumber disabled size="small" v-model="newAllocation.allocated_value" mode="currency" currency="EUR"
                     locale="de-DE" placeholder="0,00 €"></InputNumber>
      </div>
    </div>
    
    <div v-if="newSavingsCategory.savings_type === 'fixed' || isReoccurring" class="flex flex-row w-full gap-2">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringRecord.startDate.$errors[0]?.$message">
          <label>Start date</label>
        </ValidationError>
        <DatePicker size="small" v-model="newReoccurringRecord.startDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newReoccurringRecord.endDate.$errors[0]?.$message">
          <label>End date</label>
        </ValidationError>
        <DatePicker size="small" v-model="newReoccurringRecord.endDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringRecord.intervalValue.$errors[0]?.$message">
          <label>Frequency</label>
        </ValidationError>
        <InputNumber size="small" v-model="newReoccurringRecord.intervalValue" inputId="integeronly" fluid placeholder="1"></InputNumber>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringRecord.intervalUnit.$errors[0]?.$message">
          <label>Unit</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newReoccurringRecord.intervalUnit" :suggestions="filteredReoccurrenceUnits"
                        @complete="searchReoccurrenceUnit" option-label="name" placeholder="Select unit of reoccurrence" dropdown></AutoComplete>
      </div>
    </div>


    <div class="flex flex-row w-full gap-2 p-1 align-items-center">

      <div class="flex flex-column gap-2 align-items-center">
        <div class="flex flex-row w-full gap-2 p-1 align-items-center">
          <span>Interest account?</span>
          <Checkbox :value="hasInterest" @update:modelValue="toggleAccountType" binary />
        </div>
      </div>
      <div class="flex flex-column gap-2 align-items-center">
        <div v-if="newSavingsCategory.savings_type !== 'fixed'" class="flex flex-row w-full gap-2 p-1 align-items-center">
          <span>Make reoccurring?</span>
          <Checkbox :value="isReoccurring" @update:modelValue="toggleReoccurrence" binary />
        </div>
      </div>
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
            <template v-if="['goal_target'].includes(col.field)">
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

          <template v-if="!['accrued_interest'].includes(col.field)" #editor="{ data, field }">
            <template v-if="field === 'goal_target'">
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