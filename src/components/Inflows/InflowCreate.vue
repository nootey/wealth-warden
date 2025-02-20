<script setup lang="ts">

import ValidationError from "../Validation/ValidationError.vue";
import dateHelper from "../../utils/dateHelper.ts";
import {numeric, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {computed, inject, ref} from "vue";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";

const inflowStore = useInflowStore();
const toastStore = useToastStore();

const newInflow = ref(initInflow(false));

const inflowCategories = computed(() => inflowStore.inflowCategories);
const filteredInflowCategories = ref([]);

const isReoccurring = ref(false);
const reoccurringInflow = ref(initInflow(true));

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
  }
};

const v$ = useVuelidate(inflowRules, { newInflow });

function initInflow(isReoccurring: boolean): Record<string, any> {
  if (isReoccurring) {
    return {
      startDate: dateHelper.formatDate(new Date(), true),
      endDate: null,
      intervalValue: 1,
      intervalUnit: "months",
    };
  }

  return {
    amount: null,
    inflowCategory: [],
    inflowDate: dateHelper.formatDate(new Date(), true),
  };
}

async function createNewInflow() {

  v$.value.newInflow.amount.$touch();
  v$.value.newInflow.inflowDate.$touch();
  v$.value.newInflow.inflowCategory.$touch();
  if (v$.value.newInflow.$error) return;

  try {
    let inflow_date = dateHelper.mergeDateWithCurrentTime(newInflow.value.inflowDate, "Europe/Ljubljana");
    let response = await inflowStore.createInflow({
      inflow_category_id: newInflow.value.inflowCategory.id,
      inflow_category: newInflow.value.inflowCategory,
      amount: newInflow.value.amount,
      inflow_date: inflow_date});

    newInflow.value = initInflow();
    v$.value.newInflow.$reset();

    await getData();
    await getGroupedData();

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

    <div v-if="isReoccurring" class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.inflowDate.$errors[0]?.$message">
          <label>Start date</label>
        </ValidationError>
        <DatePicker v-model="reoccurringInflow.startDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.inflowDate.$errors[0]?.$message">
          <label>End date</label>
        </ValidationError>
        <DatePicker v-model="reoccurringInflow.endDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.amount.$errors[0]?.$message">
          <label>Value</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="reoccurringInflow.inputUnit" inputId="integeronly" fluid placeholder="1"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newInflow.amount.$errors[0]?.$message">
          <label>Unit</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <AutoComplete></AutoComplete>
        </InputGroup>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>