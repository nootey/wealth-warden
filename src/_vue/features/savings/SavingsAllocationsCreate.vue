<script setup lang="ts">
import {useSavingsStore} from "../../../services/stores/savingsStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, inject, ref} from "vue";
import dateHelper from "../../../utils/dateHelper.ts";
import {maxValue, minValue, numeric, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../../components/validation/ValidationError.vue";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();
const newSavingsAllocation = ref(initSavingsAllocation());

const savingsCategories = computed(() => savingsStore.savingsCategories);
const filteredSavingsCategories = ref([]);

const initData = inject<((new_page?: number | null) => Promise<void>) | null>("initData", null);

const rules = {
  newSavingsAllocation: {
    allocated_amount: {
      required,
      numeric,
      minValue: minValue(0),
      maxValue: maxValue(1000000000),
      $autoDirty: true
    },
    savingsCategory: {
      required,
      $autoDirty: true
    },
    transactionDate: {
      required,
      $autoDirty: true
    },
    description: {
      $autoDirty: true
    }
  },
};

const v$ = useVuelidate(rules, { newSavingsAllocation });

function initSavingsAllocation(): Record<string, any> {
  return {
    allocated_amount: null,
    savingsCategory: [],
    transactionDate: dateHelper.formatDate(new Date(), true),
    description: null
  };
}

async function createNewSavingsAllocation() {

  const isValidSavingsAllocation = await v$.value.newSavingsAllocation.$validate();
  if (!isValidSavingsAllocation)
    return;

  try {
    let transaction_date = dateHelper.mergeDateWithCurrentTime(newSavingsAllocation.value.transactionDate, "Europe/Ljubljana");
    let response = await savingsStore.createSavingsAllocation({
      id: null,
      transaction_type: "allocation",
      savings_category_id: newSavingsAllocation.value.savingsCategory.id,
      savings_category: newSavingsAllocation.value.savingsCategory,
      allocated_amount: newSavingsAllocation.value.allocated_amount,
      transaction_date: transaction_date,
      description: newSavingsAllocation.value.description,
    });

    newSavingsAllocation.value = initSavingsAllocation();
    v$.value.newSavingsAllocation.$reset();

    await initData();

    toastStore.successResponseToast(response);

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

const searchSavingsCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredSavingsCategories.value = [...savingsCategories.value];
    } else {
      filteredSavingsCategories.value = savingsCategories.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

</script>

<template>
  <div class="flex flex-column gap-4 p-1">

    <div class="flex flex-row w-full">
      {{ 'Add a new manual savings allocation. "Fixed" allocations will be created automatically each month' }}
    </div>
    
    <div class="flex flex-row  w-full">
      <h3>Manual allocation</h3>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsAllocation.savingsCategory.$errors[0]?.$message">
          <label>Category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-address-book"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newSavingsAllocation.savingsCategory" :suggestions="filteredSavingsCategories"
                        @complete="searchSavingsCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsAllocation.allocated_amount.$errors[0]?.$message">
          <label>Amount</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newSavingsAllocation.allocated_amount" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsAllocation.transactionDate.$errors[0]?.$message">
          <label>Date</label>
        </ValidationError>
        <DatePicker v-model="newSavingsAllocation.transactionDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewSavingsAllocation" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="false" :message="v$.newSavingsAllocation.description.$errors[0]?.$message">
          <label>Description</label>
        </ValidationError>
        <InputText size="small" v-model="newSavingsAllocation.description"></InputText>
      </div>
    </div>

    <div v-if="newSavingsAllocation.savingsCategory?.savings_type === 'fixed'" class="flex flex-row w-full gap-2 align-items-center">
      <i style="color: darkorange" class="pi pi-exclamation-triangle"></i>
      <small style="color: darkorange">{{ "The selected category has a fixed type, meaning a new allocation will be auto created each month." }}</small>
    </div>

  </div>
</template>

<style scoped>

</style>