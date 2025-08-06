<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {computed, ref} from "vue";
import type {Category, Transaction} from "../../../models/transaction_models.ts";
import {maxValue, minValue, numeric, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";

const shared_store = useSharedStore();
const toast_store = useToastStore();
const transactionStore = useTransactionStore();

const newRecord = ref<Transaction>(initData());
const allCategories = computed<Category[]>(() => transactionStore.categories);
const parentCategories = allCategories.value.filter((category) => category.parent_id == null);

const rules = {
  newRecord: {
    account_id: {
      required,
      $autoDirty: true
    },
    category_id: {
        required,
        $autoDirty: true
    },
    transaction_type: {
        required,
        $autoDirty: true
    },
    amount: {
      required,
      $autoDirty: true,
    },
    txn_date: {
      required,
      $autoDirty: true,
    },
    description: {
      required,
      $autoDirty: true,
    }
  },
};

const v$ = useVuelidate(rules, { newRecord });

const emit = defineEmits<{
  (event: 'addTransaction'): void;
}>();

function initData(): Transaction {

  return {
    id: null,
    account_id: null,
    category_id: null,
    transaction_type: "Expense",
    amount: null,
    txn_date: null,
    description: null,
  };
}

async function isRecordValid() {
  const isValid = await v$.value.newRecord.$validate();
  if (!isValid) return false;
  return true;
}

async function createNewRecord() {

  if (!await isRecordValid()) return;

  try {
    let response = await shared_store.createRecord(
        "transactions",
        {

        }
    );

    newRecord.value = initData();
    v$.value.newRecord.$reset();

    toast_store.successResponseToast(response);

    emit("addTransaction")

  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

</script>

<template>

  <div class="flex flex-column gap-3 p-1">
    <div class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column w-50">
        <SelectButton style="font-size: 0.875rem;" size="small" v-model="newRecord.transaction_type"
                      :options="parentCategories" optionLabel="name" />
      </div>
    </div>
  </div>

</template>

<style scoped>

</style>