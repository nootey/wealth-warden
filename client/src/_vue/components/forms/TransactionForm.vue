<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import type {
  Category,
  Transaction,
  Transfer,
} from "../../../models/transaction_models.ts";
import { required } from "@vuelidate/validators";
import {
  decimalValid,
  decimalMin,
  decimalMax,
} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import type { Account } from "../../../models/account_models.ts";
import dayjs from "dayjs";
import dateHelper from "../../../utils/date_helper.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import TransferForm from "./TransferForm.vue";
import ShowLoading from "../base/ShowLoading.vue";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import type { UserSettings } from "../../../models/settings_models.ts";

dayjs.extend(utc);
dayjs.extend(timezone);

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeTxOperation"): void;
  (event: "completeTrOperation"): void;
  (event: "completeTxDelete"): void;
}>();

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();
const settingsStore = useSettingsStore();

const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const defaultPreSelected = ref(false);
const userSettings = ref<UserSettings>();

const isGlobalReadOnly = computed(
  () => !!record.value.deleted_at || !!record.value.is_adjustment,
);

const isAccountRestricted = computed<boolean>(() => {
  const acc = record.value.account as Account | null | undefined;
  return (
    !!acc && typeof acc === "object" && (!!acc.closed_at || !acc.is_active)
  );
});

const isFormReadOnly = computed<boolean>(
  () => isGlobalReadOnly.value || isAccountRestricted.value,
);

const isAccountPickerDisabled = computed<boolean>(() => isGlobalReadOnly.value);

const isTxnDeleted = computed(() => !!record.value.deleted_at);
const isAccountDeleted = computed(() => !!record.value.account?.closed_at);
const isAccountActive = computed(() => !!record.value.account?.is_active);

const canRestore = computed(
  () =>
    isFormReadOnly.value &&
    isTxnDeleted.value &&
    !isAccountDeleted.value &&
    isAccountActive.value,
);

const showCantRestore = computed(
  () => isFormReadOnly.value && isTxnDeleted.value && !canRestore.value,
);

const isTransferSelected = computed(
  () => (selectedParentCategory.value?.name ?? "").toLowerCase() === "transfer",
);

const accounts = ref<Account[]>([]);
const transfer = ref<Transfer>({
  source_id: null,
  destination_id: null,
  amount: null,
  notes: null,
  deleted_at: null,
  created_at: null,
  from: null,
  to: null,
});
const transferFormRef = ref<InstanceType<typeof TransferForm> | null>(null);

const record = ref<Transaction>(initData());
const amountRef = computed({
  get: () => record.value.amount,
  set: (v) => (record.value.amount = v),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const allCategories = computed<Category[]>(() => transactionStore.categories);
const parentCategories = computed(() => {
  const base = allCategories.value.filter(
    (c) => c.display_name === "Expense" || c.display_name === "Income",
  );

  if (props.mode === "update") {
    return base;
  }

  return [
    ...base,
    {
      id: -1,
      name: "transfer",
      display_name: "Transfer",
      classification: "Transfer",
      parent_id: null,
    } as Category,
  ];
});

const selectedParentCategory = ref<Category | null>(
  parentCategories.value.find((cat) => cat.name === "expense") || null,
);

const availableCategories = computed<Category[]>(() => {
  return allCategories.value.filter(
    (category) => category.parent_id === selectedParentCategory.value?.id,
  );
});

const filteredCategories = ref<Category[]>([]);
const filteredAccounts = ref<Account[]>([]);

const rules = {
  record: {
    category: {
      name: {
        $autoDirty: true,
      },
    },
    account: {
      name: {
        required,
        $autoDirty: true,
      },
    },
    transaction_type: {
      required,
      $autoDirty: true,
    },
    amount: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
      $autoDirty: true,
    },
    txn_date: {
      required,
      $autoDirty: true,
    },
    description: {
      $autoDirty: true,
    },
  },
};

const v$ = useVuelidate(rules, { record });

onMounted(async () => {
  // Fetch accounts first
  accounts.value = accountStore.accounts;
  await getSettings();

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  } else if (props.mode === "create") {
    // Pre-select default checking account
    const defaultChecking = accounts.value.find(
      (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
    );
    if (defaultChecking) {
      record.value.account = defaultChecking;
      record.value.account_id = defaultChecking.id;
      defaultPreSelected.value = true;
    }
  }
});

async function getSettings() {
  try {
    const res = await settingsStore.getUserSettings();
    userSettings.value = res.data;
  } catch (e) {
    toastStore.errorResponseToast(e);
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
      closed_at: null,
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
      },
    },
    transaction_type: "Expense",
    amount: null,
    txn_date: dayjs().toDate(),
    description: null,
    deleted_at: null,
    is_adjustment: false,
  };
}

const todayInUserTimezone = computed(() => {
  try {
    const tz = userSettings.value?.timezone;
    if (!tz) {
      // Fallback to browser's local timezone if settings not loaded
      return dayjs().startOf("day").toDate();
    }
    return dayjs().tz(tz).startOf("day").toDate();
  } catch (error) {
    // If timezone is invalid or dayjs fails, fallback to browser local time
    console.warn(
      "Failed to calculate date in user timezone, using local:",
      error,
    );
    return dayjs().startOf("day").toDate();
  }
});

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
        return record.display_name
          .toLowerCase()
          .startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
};

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
};

async function isRecordValid() {
  const isValid = await v$.value.record.$validate();
  if (!isValid) return false;
  return true;
}

async function loadRecord(id: number) {
  try {
    loading.value = true;
    const data = await sharedStore.getRecordByID("transactions", id, {
      deleted: true,
    });

    record.value = {
      ...initData(),
      ...data,
      txn_date: data.txn_date
        ? dayjs(data.txn_date).toDate()
        : dayjs().toDate(),
    };

    selectedParentCategory.value =
      parentCategories.value.find(
        (p) =>
          p.classification?.toLowerCase?.() ===
            String(data.transaction_type).toLowerCase() ||
          p.name?.toLowerCase?.() ===
            String(data.transaction_type).toLowerCase(),
      ) || null;

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function manageRecord() {
  if (isFormReadOnly.value) {
    toastStore.infoResponseToast({
      title: "Not allowed",
      message: "This record is read only!",
    });
    return;
  }

  if (selectedParentCategory.value == null) {
    return;
  }

  if (selectedParentCategory.value.name.toLowerCase() == "transfer") {
    await startTransferOperation();
  } else {
    if (!(await isRecordValid())) return;
    await startTransactionOperation();
  }
}

async function startTransactionOperation() {
  const txn_date = dateHelper.mergeDateWithCurrentTime(
    dayjs(record.value.txn_date).format("YYYY-MM-DD"),
  );
  const recordData = {
    account_id: record.value.account.id,
    category_id: record.value.category?.id,
    transaction_type: selectedParentCategory.value?.classification,
    amount: record.value.amount,
    txn_date: txn_date,
    description: record.value.description,
  };

  try {
    let response = null;

    switch (props.mode) {
      case "create":
        response = await sharedStore.createRecord(
          transactionStore.apiPrefix,
          recordData,
        );
        break;
      case "update":
        response = await sharedStore.updateRecord(
          transactionStore.apiPrefix,
          record.value.id!,
          recordData,
        );
        break;
      default:
        emit("completeTxOperation");
        break;
    }

    // record.value = initData();
    v$.value.record.$reset();
    toastStore.successResponseToast(response);
    emit("completeTxOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function startTransferOperation() {
  const isValid = await transferFormRef.value?.v$.$validate();
  if (!isValid) return;

  const created_at = dateHelper.mergeDateWithCurrentTime(
    dayjs(transfer.value.created_at).format("YYYY-MM-DD"),
  );
  const recordData = {
    source_id: transfer.value.source_id,
    destination_id: transfer.value.destination_id,
    amount: transfer.value.amount,
    notes: transfer.value.notes,
    created_at: created_at,
  };

  try {
    const response = await transactionStore.startTransfer(recordData);
    toastStore.successResponseToast(response);
    v$.value.record.$reset();
    emit("completeTrOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function restoreTransaction() {
  try {
    let response = await transactionStore.restoreTransaction(props.recordId!);

    v$.value.record.$reset();
    toastStore.successResponseToast(response);
    emit("completeTxOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function deleteConfirmation(id: number, tx_type: string) {
  const txt = tx_type === "transfer" ? tx_type : "transaction";
  confirm.require({
    header: "Delete record?",
    message: `This will delete transaction: "${txt} : ${id}".`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id, tx_type),
  });
}

async function deleteRecord(id: number, tx_type: string) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord(
      tx_type === "transfer" ? "transactions/transfers" : "transactions",
      id,
    );
    toastStore.successResponseToast(response);
    emit("completeTxDelete");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}
</script>

<template>
  <div v-if="!loading" class="flex flex-column gap-3 p-1">
    <div
      v-if="!isFormReadOnly"
      class="flex flex-row w-full justify-content-center"
    >
      <div class="flex flex-column w-50">
        <SelectButton
          v-model="selectedParentCategory"
          style="font-size: 0.875rem"
          size="small"
          :options="parentCategories"
          option-label="display_name"
          :allow-empty="false"
          @update:model-value="updateSelectedParentCategory($event)"
        />
      </div>
    </div>
    <div v-else>
      <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
    </div>

    <h5 v-if="defaultPreSelected" style="color: var(--text-secondary)">
      Default checking account pre-selected.
    </h5>

    <div
      v-if="isTransferSelected && !isFormReadOnly"
      class="flex flex-column gap-3"
    >
      <TransferForm
        ref="transferFormRef"
        v-model:transfer="transfer"
        :accounts="accounts"
      />
    </div>

    <div v-else class="flex flex-column gap-3">
      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.account.name.$errors[0]?.$message"
          >
            <label>Account</label>
          </ValidationError>
          <AutoComplete
            v-model="record.account"
            :readonly="isAccountPickerDisabled || isFormReadOnly"
            :disabled="isAccountPickerDisabled || isFormReadOnly"
            size="small"
            :suggestions="filteredAccounts"
            option-label="name"
            force-selection
            placeholder="Select account"
            dropdown
            @complete="searchAccount"
            @update:model-value="defaultPreSelected = false"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.amount.$errors[0]?.$message"
          >
            <label>Amount</label>
          </ValidationError>
          <InputNumber
            v-model="amountNumber"
            :readonly="isFormReadOnly"
            :disabled="isFormReadOnly"
            size="small"
            mode="currency"
            currency="EUR"
            locale="de-DE"
            placeholder="0,00 €"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="false"
            :message="v$.record.category.name.$errors[0]?.$message"
          >
            <label>Category</label>
          </ValidationError>
          <AutoComplete
            v-model="record.category"
            :readonly="isFormReadOnly"
            :disabled="isFormReadOnly"
            size="small"
            :suggestions="filteredCategories"
            option-label="display_name"
            placeholder="Select category"
            dropdown
            @complete="searchCategory"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.txn_date.$errors[0]?.$message"
          >
            <label>Date</label>
          </ValidationError>
          <DatePicker
            v-model="record.txn_date"
            date-format="dd/mm/yy"
            show-icon
            fluid
            icon-display="input"
            size="small"
            :readonly="isFormReadOnly"
            :disabled="isFormReadOnly"
            :max-date="todayInUserTimezone"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="false"
            :message="v$.record.description.$errors[0]?.$message"
          >
            <label>Description</label>
          </ValidationError>
          <InputText
            v-model="record.description"
            :readonly="isFormReadOnly"
            :disabled="isFormReadOnly"
            size="small"
            placeholder="Describe transaction"
          />
        </div>
      </div>
    </div>

    <div v-if="!record.is_adjustment" class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full gap-2">
        <Button
          v-if="!isFormReadOnly"
          class="main-button"
          :label="
            selectedParentCategory?.name.toLowerCase() == 'transfer'
              ? 'Start transfer'
              : (mode == 'create' ? 'Add' : 'Update') + ' transaction'
          "
          style="height: 42px"
          @click="manageRecord"
        />
        <Button
          v-else-if="canRestore"
          class="main-button"
          label="Restore"
          style="height: 42px"
          @click="restoreTransaction"
        />
        <Button
          v-if="!isFormReadOnly && mode == 'update'"
          label="Delete transaction"
          class="delete-button"
          style="height: 42px"
          @click="deleteConfirmation(record.id!, record.transaction_type)"
        />
        <h5 v-else-if="showCantRestore" style="color: var(--text-secondary)">
          Transaction can not be restored!
        </h5>
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped></style>
