<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {computed, onMounted, ref} from "vue";
import type {Category, Transaction, Transfer} from "../../../models/transaction_models.ts";
import {required} from "@vuelidate/validators";
import {decimalValid, decimalMin, decimalMax} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import type {Account} from "../../../models/account_models.ts";
import dayjs from "dayjs";
import dateHelper from "../../../utils/date_helper.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import TransferForm from "./TransferForm.vue";
import toastHelper from "../../../utils/toast_helper.ts";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const readOnly = ref(false);

const accounts = computed<Account[]>(() => accountStore.accounts);
const transfer = ref<Transfer>({
    source_id: null,
    destination_id: null,
    amount: null,
    notes: null,
    deleted_at: null,
});
const transferFormRef = ref<InstanceType<typeof TransferForm> | null>(null);

const record = ref<Transaction>(initData());
const amountRef = computed({
  get: () => record.value.amount,
  set: v => record.value.amount = v
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const allCategories = computed<Category[]>(() => transactionStore.categories);
const parentCategories = computed(() => {
    const base = allCategories.value.filter(c =>
        c.name === "Expense" || c.name === "Income"
    );

    if(props.mode === "update") {
        return base
    }

    return [
        ...base,
        {
            id: -1,
            name: "Transfer",
            classification: "Transfer",
            parent_id: null
        } as Category
    ];
});

const selectedParentCategory = ref<Category | null>(
    parentCategories.value.find(cat => cat.name === "Expense") || null
);

const availableCategories = computed<Category[]>(() => {
  return allCategories.value.filter(
      (category) => category.parent_id === selectedParentCategory.value?.id
  );
});

const filteredCategories = ref<Category[]>([]);
const filteredAccounts = ref<Account[]>([]);

const rules = {
  record: {
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

const v$ = useVuelidate(rules, { record });

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
        sub_type: "",
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
    deleted_at: null,
    is_adjustment: false,
  };
}

async function loadRecord(id: number) {
  try {
    const data = await transactionStore.getTransactionByID(id, true);

    readOnly.value = !!data?.deleted_at || data.is_adjustment

    record.value = {
      ...initData(),
      ...data,
      txn_date: data.txn_date ? dayjs(data.txn_date).toDate() : dayjs().toDate(),
    };

    selectedParentCategory.value =
        parentCategories.value.find(
            p =>
                (p.classification?.toLowerCase?.() === String(data.transaction_type).toLowerCase()) ||
                (p.name?.toLowerCase?.() === String(data.transaction_type).toLowerCase())
        ) || null;

  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function isRecordValid() {
  const isValid = await v$.value.record.$validate();
  if (!isValid) return false;
  return true;
}

async function manageRecord() {

    if (readOnly.value) {
        toastStore.infoResponseToast(toastHelper.formatInfoToast("Not allowed", "This record is read only!"))
        return;
    }

    if (selectedParentCategory.value == null) {
    return;
    }
    
    if (selectedParentCategory.value.name.toLowerCase() == "transfer") {
        await startTransferOperation();
    } else {
        if (!await isRecordValid()) return;
        await startTransactionOperation();
    }

}

async function startTransactionOperation() {
    const txn_date = dateHelper.mergeDateWithCurrentTime(dayjs(record.value.txn_date).format('YYYY-MM-DD'))
    const recordData = {
        account_id: record.value.account.id,
        category_id: record.value.category?.id,
        transaction_type: selectedParentCategory.value?.classification,
        amount: record.value.amount,
        txn_date: txn_date,
        description: record.value.description,
    }

    try {

        let response = null;

        switch (props.mode) {
            case "create":
                response = await sharedStore.createRecord(
                    transactionStore.apiPrefix,
                    recordData
                );
                break;
            case "update":
                response = await sharedStore.updateRecord(
                    transactionStore.apiPrefix,
                    record.value.id!,
                    recordData
                );
                break;
            default:
                emit("completeOperation")
                break;
        }

        // record.value = initData();
        v$.value.record.$reset();
        toastStore.successResponseToast(response);
        emit("completeOperation")

    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

async function startTransferOperation() {
    const isValid = await transferFormRef.value?.v$.$validate();
    if (!isValid) return;

    try {
        const response = await transactionStore.startTransfer(transfer.value);
        toastStore.successResponseToast(response);
        v$.value.record.$reset();
        emit("completeOperation");
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

function updateSelectedParentCategory($event: any) {
  if ($event) {
    selectedParentCategory.value = $event;
    record.value.category = null;
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

async function restoreTransaction() {

    try {

        let response = await transactionStore.restoreTransaction(
            props.recordId!
        );

        v$.value.record.$reset();
        toastStore.successResponseToast(response);
        emit("completeOperation")

    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

</script>

<template>

  <div class="flex flex-column gap-3 p-1">

      <div v-if="!readOnly" class="flex flex-row w-full justify-content-center">
          <div class="flex flex-column w-50">
                <SelectButton style="font-size: 0.875rem;" size="small"
                              v-model="selectedParentCategory"
                              :options="parentCategories" optionLabel="name" :allowEmpty="false"
                              @update:modelValue="updateSelectedParentCategory($event)" />
          </div>
      </div>
      <div v-else>
          <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
      </div>

      <div class="flex flex-column gap-3" v-if="selectedParentCategory?.name.toLowerCase() == 'transfer' && !readOnly">
          <TransferForm ref="transferFormRef" v-model:transfer="transfer" :accounts="accounts" />
      </div>

      <div class="flex flex-column gap-3" v-else>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.account.name.$errors[0]?.$message">
                      <label>Account</label>
                  </ValidationError>
                  <AutoComplete :disabled="readOnly" size="small" v-model="record.account" :suggestions="filteredAccounts"
                                @complete="searchAccount" optionLabel="name"
                                placeholder="Select account" dropdown>
                  </AutoComplete>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.amount.$errors[0]?.$message">
                      <label>Amount</label>
                  </ValidationError>
                  <InputNumber :disabled="readOnly" size="small" v-model="amountNumber" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"></InputNumber>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="false" :message="v$.record.category.name.$errors[0]?.$message">
                      <label>Category</label>
                  </ValidationError>
                  <AutoComplete :disabled="readOnly" size="small" v-model="record.category" :suggestions="filteredCategories"
                                @complete="searchCategory" optionLabel="name"
                                placeholder="Select category" dropdown>
                  </AutoComplete>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.txn_date.$errors[0]?.$message">
                      <label>Date</label>
                  </ValidationError>
                  <DatePicker v-model="record.txn_date" date-format="dd/mm/yy"
                              showIcon fluid iconDisplay="input"
                              size="small" :disabled="readOnly"/>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="false" :message="v$.record.description.$errors[0]?.$message">
                      <label>Description</label>
                  </ValidationError>
                  <InputText :disabled="readOnly" size="small" v-model="record.description" placeholder="Describe transaction"></InputText>
              </div>
          </div>

      </div>

      <div v-if="!record.is_adjustment" class="flex flex-row gap-2 w-full" >
          <div class="flex flex-column w-full">
              <Button v-if="!readOnly" class="main-button"
                      :label="(selectedParentCategory?.name.toLowerCase() == 'transfer' ? 'Start transfer' :
                      (mode == 'create' ? 'Add' : 'Update') +  ' transaction')"
                      @click="manageRecord" style="height: 42px;" />
              <Button v-else class="main-button"
                      label="Restore"
                      @click="restoreTransaction" style="height: 42px;" />
          </div>
      </div>



  </div>

</template>

<style scoped>

</style>