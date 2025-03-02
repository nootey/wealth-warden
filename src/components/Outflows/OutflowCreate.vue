<script setup lang="ts">

import ValidationError from "../Validation/ValidationError.vue";
import dateHelper from "../../utils/dateHelper.ts";
import {integer, numeric, required, helpers} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {computed, inject, ref} from "vue";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";

const outflowStore = useOutflowStore();
const toastStore = useToastStore();

const newOutflow = ref(initOutflow(false));

const outflowCategories = computed(() => outflowStore.outflowCategories);
const filteredOutflowCategories = ref([]);
const reoccurrenceUnits = ref([
  {name: "Days"},
  {name: "Weeks"},
  {name: "Months"},
  {name: "Year"},
])
const filteredReoccurrenceUnits = ref([]);

const isReoccurring = ref(false);
const newReoccurringOutflow = ref(initOutflow(true));

const initData = inject<((new_page?: number | null) => Promise<void>) | null>("initData", null);

const emit = defineEmits<{
  (event: 'insertReoccurringActionEvent'): void;
}>();

const isEndDateValid = (value: string | null) => {
  if (!value) return true; // Allow null values
  return new Date(value) > new Date(newReoccurringOutflow.value?.startDate);
};

const outflowRules = {
  newOutflow: {
    amount: {
      required,
      numeric,
      minValue: 0,
      maxValue: 1000000000,
      $autoDirty: true
    },
    outflowCategory: {
      required,
      $autoDirty: true
    },
    outflowDate: {
      required,
      $autoDirty: true
    },
    description: {
      $autoDirty: true
    },
  },
  newReoccurringOutflow: {
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

const v$ = useVuelidate(outflowRules, { newOutflow, newReoccurringOutflow });

function initOutflow(isReoccurring: boolean = false): Record<string, any> {
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
    outflowCategory: [],
    outflowDate: dateHelper.formatDate(new Date(), true),
  };
}

async function validateOutflow(reoccurring = false) {
  const isValidOutflow = await v$.value.newOutflow.$validate();
  let isValidReoccurring = true;

  if (reoccurring) {
    isValidReoccurring = await v$.value.newReoccurringOutflow.$validate();
  }

  if (!isValidReoccurring) return true;
  if (!isValidOutflow) return true;

  return false;
}


async function createNewOutflow() {

  if (await validateOutflow(isReoccurring.value)) return;

  if(isReoccurring.value) return;

  try {
    let outflow_date = dateHelper.mergeDateWithCurrentTime(newOutflow.value.outflowDate, "Europe/Ljubljana");
    let response = await outflowStore.createOutflow({
      id: null,
      outflow_category_id: newOutflow.value.outflowCategory.id,
      outflow_category: newOutflow.value.outflowCategory,
      amount: newOutflow.value.amount,
      outflow_date: outflow_date,
      description: newOutflow.value.description,
    });

    newOutflow.value = initOutflow(false);
    v$.value.newOutflow.$reset();

    await initData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function createNewReoccurringOutflow() {

  if (await validateOutflow(true)) return;

  let outflow_date = dateHelper.mergeDateWithCurrentTime(newOutflow.value.outflowDate, "Europe/Ljubljana");
  let start_date = dateHelper.mergeDateWithCurrentTime(newReoccurringOutflow.value.start_date, "Europe/Ljubljana");
  let end_date = newReoccurringOutflow.value.end_date ? dateHelper.mergeDateWithCurrentTime(newReoccurringOutflow.value.end_date, "Europe/Ljubljana") : null;

  try {

    let response = await outflowStore.createReoccurringOutflow({
          outflow_category_id: newOutflow.value.outflowCategory.id,
          outflow_category: newOutflow.value.outflowCategory,
          amount: newOutflow.value.amount,
          outflow_date: outflow_date
        },
        {
          category_type: "outflow",
          start_date: start_date,
          end_date: end_date,
          interval_unit: newReoccurringOutflow.value.intervalUnit.name,
          interval_value: newReoccurringOutflow.value.intervalValue
        });

    newOutflow.value = initOutflow(false);
    v$.value.newOutflow.$reset();
    newReoccurringOutflow.value = initOutflow(true);
    v$.value.newReoccurringOutflow.$reset();

    emit("insertReoccurringActionEvent");
    await initData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

const searchOutflowCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredOutflowCategories.value = [...outflowCategories.value];
    } else {
      filteredOutflowCategories.value = outflowCategories.value.filter((outflowCategory) => {
        return outflowCategory.name.toLowerCase().startsWith(event.query.toLowerCase());
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
    newReoccurringOutflow.value.startDate = event;
  }
}

function toggleReoccurrence(event: any){
  if(event === true){
    if(newOutflow.value.outflowDate){
      newReoccurringOutflow.value.startDate = newOutflow.value.outflowDate;
    }
  } else {
    newReoccurringOutflow.value = initOutflow(true);
  }
  isReoccurring.value = event;
}

</script>

<template>
  <div class="flex flex-column gap-4 p-1">

    <div class="flex flex-row w-full">
      {{ "Add new outflows. Chose between a single entry, or a reoccurring outflow." }}
    </div>

    <div class="flex flex-row w-full">
      {{ "Reoccurring outflows will get added automatically, based on your set parameters. You will also receive a notification for each one." }}
    </div>

    <div class="flex flex-row  w-full">
      <h3>Single entry</h3>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflow.outflowCategory.$errors[0]?.$message">
          <label>Category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-address-book"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newOutflow.outflowCategory" :suggestions="filteredOutflowCategories"
                        @complete="searchOutflowCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflow.amount.$errors[0]?.$message">
          <label>Amount</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newOutflow.amount" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newOutflow.outflowDate.$errors[0]?.$message">
          <label>Date</label>
        </ValidationError>
        <DatePicker v-model="newOutflow.outflowDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;" @update:modelValue="updateStartDate"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button :disabled="isReoccurring" icon="pi pi-cart-plus" @click="createNewOutflow" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="false" :message="v$.newOutflow.description.$errors[0]?.$message">
          <label>Description</label>
        </ValidationError>
        <InputText size="small" v-model="newOutflow.description"></InputText>
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
      <span>  {{ "This reoccurring action will trigger on: " + (newReoccurringOutflow.startDate ? dateHelper.formatDate(newReoccurringOutflow.startDate) : "/") }}  </span>
      <span>  {{ ", and will end on: " + (newReoccurringOutflow.endDate ? dateHelper.formatDate(newReoccurringOutflow.endDate) : "until canceled") }} </span>
    </div>

    <div v-if="isReoccurring" class="flex flex-row  w-full">
      <span> {{ "It will repeat every: " + (newReoccurringOutflow.intervalValue ?? 0) + " " + (newReoccurringOutflow.intervalUnit.name ??  "times") }}  </span>
    </div>

    <div v-if="isReoccurring" class="flex flex-row w-full gap-2 p-1 align-items-center">

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringOutflow.startDate.$errors[0]?.$message">
          <label>Start date</label>
        </ValidationError>
        <DatePicker v-model="newReoccurringOutflow.startDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;" />
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" :message="v$.newReoccurringOutflow.endDate.$errors[0]?.$message">
          <label>End date</label>
        </ValidationError>
        <DatePicker v-model="newReoccurringOutflow.endDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringOutflow.intervalValue.$errors[0]?.$message">
          <label>Value</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-percentage"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newReoccurringOutflow.intervalValue" inputId="integeronly" fluid placeholder="1"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringOutflow.intervalUnit.$errors[0]?.$message">
          <label>Unit</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-ellipsis-v"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newReoccurringOutflow.intervalUnit" :suggestions="filteredReoccurrenceUnits"
                        @complete="searchReoccurrenceUnit" option-label="name" placeholder="Select unit of reoccurrence" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewReoccurringOutflow" style="height: 42px;" />
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>