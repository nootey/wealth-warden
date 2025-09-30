<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {computed, nextTick, onMounted, ref, watch} from "vue";
import type {Category, TransactionTemplate} from "../../../models/transaction_models.ts";
import {maxValue, minValue, required} from "@vuelidate/validators";
import {decimalValid, decimalMin, decimalMax} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import type {Account} from "../../../models/account_models.ts";
import dayjs from "dayjs";
import currencyHelper from "../../../utils/currency_helper.ts";
import toastHelper from "../../../utils/toast_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import vueHelper from "../../../utils/vue_helper.ts";

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

const isReadOnly = ref(false);
const isImmutable = ref(false);

const isAccountRestricted = computed<boolean>(() => {
    const acc = record.value.account as Account | null | undefined;
    return !!acc && typeof acc === 'object' && (!!acc.deleted_at || !acc.is_active);
});

const loading = ref(false);

const record = ref<TransactionTemplate>(initData());
const amountRef = computed({
  get: () => record.value.amount,
  set: v => record.value.amount = v
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const frequencies = ref<string[]>(["Weekly", "Biweekly", "Monthly", "Quarterly", "Annually"]);
const accounts = computed<Account[]>(() => accountStore.accounts);
const allCategories = computed<Category[]>(() => transactionStore.categories);
const parentCategories = computed(() => {
    const base = allCategories.value.filter(c =>
        c.display_name === "Expense" || c.display_name === "Income"
    );

    if(props.mode === "update") {
        return base
    }

    return [
        ...base
    ];
});

const selectedParentCategory = ref<Category | null>(
    parentCategories.value.find(cat => cat.name === "expense") || null
);

const availableCategories = computed<Category[]>(() => {
  return allCategories.value.filter(
      (category) => category.parent_id === selectedParentCategory.value?.id
  );
});

const filteredCategories = ref<Category[]>([]);
const filteredAccounts = ref<Account[]>([]);
const filteredFrequencies = ref<string[]>([]);

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
      name: {
          required,
          $autoDirty: true,
      },
      frequency: {
          required,
          $autoDirty: true,
      },
      next_run_at: {
          required,
          $autoDirty: true,
      },
      end_date: {
          $autoDirty: true,
      },
      max_runs: {
          $autoDirty: true,
          min: minValue(1),
          max: maxValue(99999),
      },
      is_active: {
          required,
          $autoDirty: true,
      },
  },
};

const v$ = useVuelidate(rules, { record });

function initData(): TransactionTemplate {

  return {
      id: null,
      name: "",
      account_id: null,
      category_id: null,
      category: {
          id: null,
          name: "",
          display_name: "",
          classification: "",
          is_default: true,
          parent_id: null,
          deleted_at: null,
      },
      account: {
          id: null,
          name: "",
          is_active: true,
          deleted_at: null,
          account_type: {
              id: null,
              name: "",
              type: "",
              sub_type: "",
              classification: ""
          },
          balance: {
              id: null,
              as_of: null,
              start_balance: null,
              end_balance: null
          }
      },
      transaction_type: "Expense",
      amount: null,
      period: "",
      run_count: 0,
      end_date: null,
      is_active: true,
      frequency: "",
  };
}

const tomorrowUtcMidnight = computed(() => {
    const now = new Date()
    return new Date(
        Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + 1)
    )
})

const endDateMin = computed(() => {
    if (record.value.next_run_at) {
        return dayjs(record.value.next_run_at).add(1, "day").toDate()
    }
    return tomorrowUtcMidnight.value
})

watch(
    () => record.value.next_run_at,
    (newNextRun) => {
        if (!newNextRun) {
            record.value.end_date = null
            return
        }

        if (record.value.end_date && record.value.end_date < newNextRun) {
            record.value.end_date = null
        }
    }
)

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

const searchFrequency = (event: { query: string }) => {
    setTimeout(() => {
        if (!event.query.trim().length) {
            filteredFrequencies.value = [...frequencies.value];
        } else {
            filteredFrequencies.value = frequencies.value.filter((record) => {
                return record.toLowerCase().startsWith(event.query.toLowerCase());
            });
        }
    }, 250);
}

async function isRecordValid() {
    const isValid = await v$.value.record.$validate();
    if (!isValid) return false;
    return true;
}

async function loadRecord(id: number) {
  try {
    isImmutable.value = true;
    loading.value = true;
    const data = await sharedStore.getRecordByID("transactions/templates", id);

    record.value = {
      ...initData(),
      ...data,
      frequency: vueHelper.capitalize(data.frequency),
    };

    selectedParentCategory.value =
        parentCategories.value.find(
            p =>
                (p.classification?.toLowerCase?.() === String(data.transaction_type).toLowerCase()) ||
                (p.name?.toLowerCase?.() === String(data.transaction_type).toLowerCase())
        ) || null;

    await nextTick();
    loading.value = false;

  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function manageRecord() {

    if (isReadOnly.value || isAccountRestricted.value) {
        toastStore.infoResponseToast(toastHelper.formatInfoToast("Not allowed", "This record is read only!"))
        return;
    }

    if (selectedParentCategory.value == null) {
    return;
    }

    if (!await isRecordValid()) return;
    await startOperation();

}

async function startOperation() {

    const recordData = {
        name: record.value.name,
        is_active: record.value.is_active,
        account_id: record.value.account.id,
        category_id: record.value.category?.id,
        transaction_type: selectedParentCategory.value?.classification,
        amount: record.value.amount,
        frequency: record.value.frequency,
        next_run_at: record.value.next_run_at,
        end_date: record.value.end_date,
        max_runs: record.value.max_runs,
    }

    try {

        let response = null;

        switch (props.mode) {
            case "create":
                response = await sharedStore.createRecord(
                    "transactions/templates",
                    recordData
                );
                break;
            case "update":
                response = await sharedStore.updateRecord(
                    "transactions/templates",
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

</script>

<template>

  <div v-if="!loading" class="flex flex-column gap-3 p-1">

      <div v-if="!isImmutable" class="flex flex-row w-full justify-content-center">
          <div class="flex flex-column w-50">
                <SelectButton style="font-size: 0.875rem;" size="small"
                              v-model="selectedParentCategory"
                              :options="parentCategories" optionLabel="display_name" :allowEmpty="false"
                              @update:modelValue="updateSelectedParentCategory($event)" />
          </div>
      </div>
      <div v-else>
          <h5 style="color: var(--text-secondary)">Some parts of the record are immutable.</h5>
      </div>

      <div class="flex flex-column gap-3">

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.name.$errors[0]?.$message">
                      <label>Name</label>
                  </ValidationError>
                  <InputText :readonly="isReadOnly" :disabled="isReadOnly" size="small" v-model="record.name" placeholder="Input name"></InputText>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.account.name.$errors[0]?.$message">
                      <label>Account</label>
                  </ValidationError>
                  <AutoComplete :readonly="isAccountRestricted || isReadOnly || isImmutable" :disabled="isAccountRestricted || isReadOnly || isImmutable" size="small"
                                v-model="record.account" :suggestions="filteredAccounts"
                                @complete="searchAccount" optionLabel="name" forceSelection
                                placeholder="Select account" dropdown>
                  </AutoComplete>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="true" :message="v$.record.amount.$errors[0]?.$message">
                      <label>Amount</label>
                  </ValidationError>
                  <InputNumber :readonly="isReadOnly " :disabled="isReadOnly" size="small" v-model="amountNumber" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"></InputNumber>
              </div>
          </div>

          <div class="flex flex-row w-full">
              <div class="flex flex-column gap-1 w-full">
                  <ValidationError :isRequired="false" :message="v$.record.category.name.$errors[0]?.$message">
                      <label>Category</label>
                  </ValidationError>
                  <AutoComplete :readonly="isImmutable" :disabled="isImmutable" size="small" v-model="record.category" :suggestions="filteredCategories"
                                @complete="searchCategory" optionLabel="display_name"
                                placeholder="Select category" dropdown>
                  </AutoComplete>
              </div>
          </div>

          <div class="flex flex-row w-full gap-2 align-items-center">
              <div class="flex flex-column gap-1 w-12">
                  <ValidationError :isRequired="true" :message="v$.record.frequency.$errors[0]?.$message">
                      <label>Frequency</label>
                  </ValidationError>
                  <AutoComplete :readonly="isImmutable" :disabled="isImmutable" size="small"
                                v-model="record.frequency" :suggestions="filteredFrequencies"
                                @complete="searchFrequency"
                                placeholder="Select frequency" dropdown>
                  </AutoComplete>
              </div>
          </div>

          <div class="flex flex-row w-full gap-2 align-items-center">
              <div class="flex flex-column gap-1 w-12">
                  <ValidationError :isRequired="true" :message="v$.record.frequency.$errors[0]?.$message">
                      <label>{{ mode === 'create' ? "First run" : "Next run"}}</label>
                  </ValidationError>
                  <DatePicker v-model="record.next_run_at" date-format="dd/mm/yy"
                              showIcon fluid iconDisplay="input" size="small"
                              :readonly="isReadOnly" :disabled="isReadOnly"
                              :minDate="tomorrowUtcMidnight"
                  />
              </div>
          </div>

          <div class="flex flex-row w-full gap-2">
              <div class="flex flex-column gap-1 w-6">
                  <ValidationError :isRequired="false" :message="v$.record.end_date.$errors[0]?.$message">
                      <label>End date</label>
                  </ValidationError>
                  <DatePicker v-model="record.end_date" date-format="dd/mm/yy"
                              showIcon fluid iconDisplay="input" size="small"
                              :readonly="isReadOnly" :disabled="isReadOnly"
                              :minDate="endDateMin"
                  />
              </div>
              <div class="flex flex-column gap-1 w-6">
                  <ValidationError :isRequired="false" :message="v$.record.max_runs.$errors[0]?.$message">
                      <label>Max runs</label>
                  </ValidationError>
                  <InputNumber :readonly="isReadOnly" :disabled="isReadOnly" size="small"
                               v-model="record.max_runs" placeholder="1">
                  </InputNumber>
              </div>
          </div>

      </div>

      <div class="flex flex-row gap-2 w-full" >
          <div class="flex flex-column w-full">
              <Button v-if="!isReadOnly" class="main-button"
                      :label="(mode == 'create' ? 'Add' : 'Update') +  ' template'"
                      @click="manageRecord" style="height: 42px;" />
          </div>
      </div>



  </div>
  <ShowLoading v-else :numFields="7" />

</template>

<style scoped>

</style>