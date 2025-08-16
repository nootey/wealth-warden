<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {computed, onMounted, ref, toRef} from "vue";
import type {Category, Transaction} from "../../../models/transaction_models.ts";
import {required} from "@vuelidate/validators";
import {decimalValid, decimalMin, decimalMax} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import type {Account} from "../../../models/account_models.ts";
import dayjs from "dayjs";
import dateHelper from "../../../utils/date_helper.ts";
import currencyHelper from "../../../utils/currency_helper.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

const accounts = ref<Account[]>([]);

onMounted(async () => {
  await getAccounts();
})

const newRecord = ref<Transaction>(initData());
const amountRef = toRef(newRecord.value, "amount");
const amountNumber = currencyHelper.useMoneyField(amountRef, 2).number;

const allCategories = computed<Category[]>(() => transactionStore.categories);
const parentCategories = allCategories.value.filter((category) => (category.name == "Expense") || category.name == "Income");

const selectedParentCategory = ref<Category | null>(
    parentCategories.find(cat => cat.name === "Expense") || null
);
const availableCategories = computed<Category[]>(() => {
  return allCategories.value.filter(
      (category) => category.parent_id === selectedParentCategory.value?.id
  );
});

const filteredCategories = ref<Category[]>([]);
const filteredAccounts = ref<Account[]>([]);

const rules = {
  newRecord: {
    category: {
        name: {
          $autoDirty: true
        }
    },
    account: {
      name: {
        required,
        $autoDirty: true
      }
    },
    transaction_type: {
        required,
        $autoDirty: true
    },
    amount: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
      $autoDirty: true
    },
    txn_date: {
      required,
      $autoDirty: true,
    },
    description: {
      $autoDirty: true,
    }
  },
};

const v$ = useVuelidate(rules, { newRecord });

const emit = defineEmits<{
  (event: 'addTransaction'): void;
}>();

async function getAccounts() {
  try {
    const response = await accountStore.getAllAccounts();
    accounts.value = response.data;
  } catch (e) {
    toastStore.errorResponseToast(e)
  }
}

function initData(): Transaction {

  return {
    id: null,
    account_id: null,
    category_id: null,
    category: {
      id: null,
      name: "",
      classification: "",
      parent_id: null,
    },
    account: {
      id: null,
      name: "",
      account_type: {
        id: null,
        name: "",
        type: "",
        subtype: "",
        classification: "",
      },
      balance: {
        id: null,
        as_of: null,
        start_balance: null,
        end_balance: null,
      }
    },
    transaction_type: "Expense",
    amount: null,
    txn_date: dayjs().toDate(),
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

  if (selectedParentCategory.value == null) {
    return;
  }

  const txn_date = dateHelper.mergeDateWithCurrentTime(dayjs(newRecord.value.txn_date).format('YYYY-MM-DD'))

  try {
    let response = await sharedStore.createRecord(
        "transactions",
        {
          account_id: newRecord.value.account.id,
          category_id: newRecord.value.category?.id,
          transaction_type: selectedParentCategory.value.classification,
          amount: newRecord.value.amount,
          txn_date: txn_date,
          description: newRecord.value.description,
        }
    );

    newRecord.value = initData();
    v$.value.newRecord.$reset();

    toastStore.successResponseToast(response);

    emit("addTransaction")

  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function updateSelectedParentCategory($event: any) {
  if ($event) {
    selectedParentCategory.value = $event;
    newRecord.value.category = null;
    filteredCategories.value = [];
  }
}

const searchCategory = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredCategories.value = [...availableCategories.value];
    } else {
      filteredCategories.value = availableCategories.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

const searchAccount = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredAccounts.value = [...accounts.value];
    } else {
      filteredAccounts.value = accounts.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

</script>

<template>

  <div class="flex flex-column gap-3 p-1">

    <div class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column w-50">
        <SelectButton style="font-size: 0.875rem;" size="small"
                      v-model="selectedParentCategory"
                      :options="parentCategories" optionLabel="name" :allowEmpty="false"
                      @update:modelValue="updateSelectedParentCategory($event)" />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.account.name.$errors[0]?.$message">
          <label>Account</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newRecord.account" :suggestions="filteredAccounts"
                      @complete="searchAccount" optionLabel="name"
                      placeholder="Select account" dropdown>
        </AutoComplete>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.amount.$errors[0]?.$message">
          <label>Amount</label>
        </ValidationError>
        <InputNumber size="small" v-model="amountNumber" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"></InputNumber>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="false" :message="v$.newRecord.category.name.$errors[0]?.$message">
            <label>Category</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newRecord.category" :suggestions="filteredCategories"
                      @complete="searchCategory" optionLabel="name"
                      placeholder="Select category" dropdown>
        </AutoComplete>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.txn_date.$errors[0]?.$message">
            <label>Date</label>
        </ValidationError>
        <DatePicker v-model="newRecord.txn_date" date-format="dd/mm/yy"
                    showIcon fluid iconDisplay="input"
                    size="small"/>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="false" :message="v$.newRecord.description.$errors[0]?.$message">
          <label>Description</label>
        </ValidationError>
        <InputText size="small" v-model="newRecord.description" placeholder="Describe transaction"></InputText>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full">
        <Button class="main-button" label="Add transaction" @click="createNewRecord" style="height: 42px;" />
      </div>
    </div>

  </div>

</template>

<style scoped>

</style>