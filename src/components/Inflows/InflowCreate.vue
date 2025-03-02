<script setup lang="ts">

import ValidationError from "../Validation/ValidationError.vue";
import dateHelper from "../../utils/dateHelper.ts";
import {integer, numeric, required, helpers} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {computed, inject, ref} from "vue";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const newInflow = ref(initInflow(false));

const inflowCategories = computed(() => inflowStore.inflowCategories);
const filteredInflowCategories = ref([]);
const reoccurrenceUnits = ref([
  {name: "Days"},
  {name: "Weeks"},
  {name: "Months"},
  {name: "Year"},
])
const filteredReoccurrenceUnits = ref([]);

const isReoccurring = ref(false);
const newReoccurringInflow = ref(initInflow(true));

const initData = inject<((new_page?: number | null) => Promise<void>) | null>("initData", null);

const emit = defineEmits<{
  (event: 'insertReoccurringActionEvent'): void;
}>();

const isEndDateValid = (value: string | null) => {
  if (!value) return true; // Allow null values
  return new Date(value) > new Date(newReoccurringInflow.value?.startDate);
};

const inflowRules = {
  newInflow: {
    amount: {
      required,
      numeric,
      minValue: 0,
      maxValue: 1000000000,
      $autoDirty: true
    },
    inflowCategory: {
      required,
      $autoDirty: true
    },
    inflowDate: {
      required,
      $autoDirty: true
    },
    description: {
      $autoDirty: true
    },
  },
  newReoccurringInflow: {
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
      minValue: 0,
      maxValue: 9,
      $autoDirty: true
    },
    intervalUnit: {
      required,
      $autoDirty: true
    },
  }
};

const v$ = useVuelidate(inflowRules, { newInflow, newReoccurringInflow });

function initInflow(isReoccurring: boolean = false): Record<string, any> {
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
    amount: null,
    inflowCategory: [],
    inflowDate: dateHelper.formatDate(new Date(), true),
  };
}

async function validateInflow(reoccurring = false) {
  const isValidInflow = await v$.value.newInflow.$validate();
  let isValidReoccurring = true;

  if (reoccurring) {
    isValidReoccurring = await v$.value.newReoccurringInflow.$validate();
  }

  if (!isValidReoccurring) return true;
  if (!isValidInflow) return true;

  return false;
}


async function createNewInflow() {

  if (await validateInflow(isReoccurring.value)) return;

  if(isReoccurring.value) return;

  try {
    let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");
    let response = await inflowStore.createInflow({
      id: null,
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date,
      description: newInflow.value.description,
    });

    newInflow.value = initInflow(false);
    v$.value.newInflow.$reset();

    await initData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function createNewReoccurringInflow() {

  if (await validateInflow(true)) return;

  let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");
  let start_date = dateHelper.mergeDateWithCurrentTime(newReoccurringInflow.value.start_date, "Europe/Ljubljana");
  let end_date = newReoccurringInflow.value.end_date ? dateHelper.mergeDateWithCurrentTime(newReoccurringInflow.value.end_date, "Europe/Ljubljana") : null;

  try {

    let response = await inflowStore.createReoccurringInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date
      },
      {
      category_type: "inflow",
      start_date: start_date,
      end_date: end_date,
      interval_unit: newReoccurringInflow.value.intervalUnit.name,
      interval_value: newReoccurringInflow.value.intervalValue
      });

    newInflow.value = initInflow(false);
    v$.value.newInflow.$reset();
    newReoccurringInflow.value = initInflow(true);
    v$.value.newReoccurringInflow.$reset();

    emit("insertReoccurringActionEvent");
    await initData();

    toastStore.successResponseToast(response);

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

function updateStartDate(event: any) {
  if (isReoccurring.value){
    newReoccurringInflow.value.startDate = event;
  }
}

function toggleReoccurrence(event: any){
  if(event === true){
    if(newInflow.value.inflowDate){
      newReoccurringInflow.value.startDate = newInflow.value.inflowDate;
    }
  } else {
    newReoccurringInflow.value = initInflow(true);
  }
  isReoccurring.value = event;
}

</script>

<template>
  <div class="flex flex-column gap-4 p-1">

    <div class="flex flex-row w-full">
      {{ "Add new inflows. Chose between a single entry, or a reoccurring inflow." }}
    </div>

    <div class="flex flex-row w-full">
      {{ "Reoccurring inflows will get added automatically, based on your set parameters. You will also receive a notification for each one." }}
    </div>

    <div class="flex flex-row  w-full">
      <h3>Single entry</h3>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.inflowCategory.$errors[0]?.$message">
          <label>Category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-address-book"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newInflow.inflowCategory" :suggestions="filteredInflowCategories"
                        @complete="searchInflowCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.amount.$errors[0]?.$message">
          <label>Amount</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newInflow.amount" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.inflowDate.$errors[0]?.$message">
          <label>Date</label>
        </ValidationError>
        <DatePicker v-model="newInflow.inflowDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;" @update:modelValue="updateStartDate"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button :disabled="isReoccurring" icon="pi pi-cart-plus" @click="createNewInflow" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="false" :message="v$.newInflow.description.$errors[0]?.$message">
          <label>Description</label>
        </ValidationError>
        <InputText size="small" v-model="newInflow.description"></InputText>
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <span>Make reoccurring?</span>
      <Checkbox :value="isReoccurring" @update:modelValue="toggleReoccurrence" binary />
    </div>

    <div v-if="isReoccurring" class="flex flex-row  w-full">
      <h3>Reoccurring details</h3>
    </div>

    <div v-if="isReoccurring" class="flex flex-row  w-full">
      <span>  {{ "This reoccurring action will trigger on: " + (newReoccurringInflow.startDate ? dateHelper.formatDate(newReoccurringInflow.startDate) : "/") }}  </span>
      <span>  {{ ", and will end on: " + (newReoccurringInflow.endDate ? dateHelper.formatDate(newReoccurringInflow.endDate) : "until canceled") }} </span>
    </div>

    <div v-if="isReoccurring" class="flex flex-row  w-full">
      <span> {{ "It will repeat every: " + (newReoccurringInflow.intervalValue ?? 0) + " " + (newReoccurringInflow.intervalUnit.name ??  "times") }}  </span>
    </div>

    <div v-if="isReoccurring" class="flex flex-row w-full gap-2 p-1 align-items-center">

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringInflow.startDate.$errors[0]?.$message">
          <label>Start date</label>
        </ValidationError>
        <DatePicker v-model="newReoccurringInflow.startDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;" />
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newReoccurringInflow.endDate.$errors[0]?.$message">
          <label>End date</label>
        </ValidationError>
        <DatePicker v-model="newReoccurringInflow.endDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringInflow.intervalValue.$errors[0]?.$message">
          <label>Value</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-percentage"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newReoccurringInflow.intervalValue" inputId="integeronly" fluid placeholder="1"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringInflow.intervalUnit.$errors[0]?.$message">
          <label>Unit</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-ellipsis-v"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newReoccurringInflow.intervalUnit" :suggestions="filteredReoccurrenceUnits"
                        @complete="searchReoccurrenceUnit" option-label="name" placeholder="Select unit of reoccurrence" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewReoccurringInflow" style="height: 42px;" />
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>