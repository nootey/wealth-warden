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
const newSavingsDeduction = ref(initSavingsDeduction());

const savingsCategories = computed(() => savingsStore.savingsCategories);
const filteredSavingsCategories = ref([]);

const initData = inject<((new_page?: number | null) => Promise<void>) | null>("initData", null);

const rules = {
  newSavingsDeduction: {
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

const v$ = useVuelidate(rules, { newSavingsDeduction });

function initSavingsDeduction(): Record<string, any> {
  return {
    allocated_amount: null,
    savingsCategory: [],
    transactionDate: dateHelper.formatDate(new Date(), true),
    description: null,
  };
}

async function createNewSavingsDeduction() {

  const isValidSavingsDeduction = await v$.value.newSavingsDeduction.$validate();
  if (!isValidSavingsDeduction)
    return;

  try {
    let transaction_date = dateHelper.mergeDateWithCurrentTime(newSavingsDeduction.value.transactionDate, "Europe/Ljubljana");
    let response = await savingsStore.createSavingsDeduction({
      id: null,
      transaction_type: "deduction",
      savings_category_id: newSavingsDeduction.value.savingsCategory.id,
      savings_category: newSavingsDeduction.value.savingsCategory,
      allocated_amount: newSavingsDeduction.value.allocated_amount,
      transaction_date: transaction_date,
      description: newSavingsDeduction.value.description,
    });

    newSavingsDeduction.value = initSavingsDeduction();
    v$.value.newSavingsDeduction.$reset();

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
      {{ 'Add a new savings deduction.' }}
    </div>

    <div class="flex flex-row  w-full">
      <h3>New deduction</h3>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsDeduction.savingsCategory.$errors[0]?.$message">
          <label>Category</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-address-book"></i>
          </InputGroupAddon>
          <AutoComplete size="small" v-model="newSavingsDeduction.savingsCategory" :suggestions="filteredSavingsCategories"
                        @complete="searchSavingsCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsDeduction.allocated_amount.$errors[0]?.$message">
          <label>Amount</label>
        </ValidationError>
        <InputGroup>
          <InputGroupAddon>
            <i class="pi pi-wallet"></i>
          </InputGroupAddon>
          <InputNumber size="small" v-model="newSavingsDeduction.allocated_amount" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
        </InputGroup>
      </div>

      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.newSavingsDeduction.transactionDate.$errors[0]?.$message">
          <label>Date</label>
        </ValidationError>
        <DatePicker v-model="newSavingsDeduction.transactionDate" date-format="dd/mm/yy" showIcon fluid iconDisplay="input"
                    style="height: 42px;"/>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Submit</label>
        </ValidationError>
        <Button icon="pi pi-cart-plus" @click="createNewSavingsDeduction" style="height: 42px;" />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2 p-1 align-items-center">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="false" :message="v$.newSavingsDeduction.description.$errors[0]?.$message">
          <label>Reason</label>
        </ValidationError>
        <InputText size="small" v-model="newSavingsDeduction.description"></InputText>
      </div>
    </div>

  </div>
</template>

<style scoped>

</style>