<script setup lang="ts">

import ValidationError from "../Validation/ValidationError.vue";
import dateHelper from "../../utils/dateHelper.ts";
import {integer, numeric, required} from "@vuelidate/validators";
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

const getData = inject<((new_page?: number | null) => Promise<void>) | null>("getData", null);
const getGroupedData = inject<(() => Promise<void>) | null>("getGroupedData", null);

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
  },
  newReoccurringInflow: {
    startDate: {
      required,
      $autoDirty: true
    },
    endDate: {
      $autoDirty: true
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

  try {
    let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");
    let response = await inflowStore.createInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date});

    newInflow.value = initInflow(false);
    v$.value.newInflow.$reset();

    await getData();
    await getGroupedData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function createNewReoccurringInflow() {

  if (await validateInflow(true)) return;

  let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");

  try {
    let response = await inflowStore.createReoccurringInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date
      },
      {
      startDate: newReoccurringInflow.value.startDate,
      endDate: newReoccurringInflow.value.endDate,
      intervalUnit: newReoccurringInflow.value.intervalUnit,
      intervalValue: newReoccurringInflow.value.intervalValue
      });

    newInflow.value = initInflow(false);
    v$.value.newInflow.$reset();
    newReoccurringInflow.value = initInflow(true);
    v$.value.newReoccurringInflow.$reset();

    // await getData();
    // await getGroupedData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function editInflow(id: number) {
  console.log(id)
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
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewInflow" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <span>Make reoccurring?</span>
      <Checkbox v-model="isReoccurring" binary />
    </div>

    <div v-if="isReoccurring" class="flex flex-row  w-full">
      <h3>Reoccurring details</h3>
    </div>

    <div v-if="isReoccurring" class="flex flex-row w-full gap-2 p-1 align-items-center">

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newReoccurringInflow.startDate.$errors[0]?.$message">
          <label>Start date</label>
        </ValidationError>
        <DatePicker v-model="newReoccurringInflow.startDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
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