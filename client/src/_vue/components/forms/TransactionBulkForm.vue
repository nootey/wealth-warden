<script setup lang="ts">
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { computed, onMounted, ref, watch } from "vue";
import { required } from "@regle/rules";
import { useRegle } from "@regle/core";
import {
  decimalValid,
  decimalMin,
  decimalMax,
} from "../../../validators/currency.ts";
import Decimal from "decimal.js";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import dateHelper from "../../../utils/date_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import searchHelper from "../../../utils/search_helper.ts";
import ValidationError from "../validation/ValidationError.vue";
import AvailableBalance from "../base/AvailableBalance.vue";
import BaseForm from "../base/BaseForm.vue";
import TransactionBulkRow from "./TransactionBulkRow.vue";
import type { Account } from "../../../models/account_models.ts";
import type {
  BulkTransactionRow,
  Category,
} from "../../../models/transaction_models.ts";
import type { UserSettings } from "../../../models/settings_models.ts";

dayjs.extend(utc);
dayjs.extend(timezone);

const emit = defineEmits<{
  (event: "completeTxOperation"): void;
}>();

// Matches the backend cap on rows per request.
const MAX_ROWS = 50;

const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();
const settingsStore = useSettingsStore();

const sharedDirection = defineModel<boolean>("sharedDirection", {
  default: true,
});
const sharedDescription = defineModel<boolean>("sharedDescription", {
  default: false,
});

const submitting = ref(false);
const defaultPreSelected = ref(false);
const userSettings = ref<UserSettings>();

type BulkFormData = {
  account: Account | null;
  txn_date: Date;
  description: string | null;
  rows: BulkTransactionRow[];
};

const parentCategories = computed(() =>
  transactionStore.categories.filter(
    (c) => c.display_name === "Expense" || c.display_name === "Income",
  ),
);

const selectedParentCategory = ref<Category | null>(
  parentCategories.value.find((c) => c.name === "expense") ?? null,
);

const record = ref<BulkFormData>({
  account: null,
  txn_date: dayjs().toDate(),
  description: null,
  rows: [newRow()],
});

const accounts = computed<Account[]>(() => accountStore.accounts);
const filteredAccounts = ref<Account[]>([]);

const categoriesByDirection = computed<Record<string, Category[]>>(() =>
  Object.fromEntries(
    parentCategories.value.map((parent) => [
      parent.classification,
      transactionStore.categories.filter((c) => c.parent_id === parent.id),
    ]),
  ),
);

const hasExpenseRow = computed(() =>
  record.value.rows.some((row) => row.direction === "expense"),
);

const total = computed(() =>
  record.value.rows.reduce((sum, row) => {
    const amount = new Decimal(row.amount ?? 0);
    return !sharedDirection.value && row.direction === "expense"
      ? sum.minus(amount)
      : sum.plus(amount);
  }, new Decimal(0)),
);

const submitLabel = computed(() => {
  const count = record.value.rows.length;
  return `Add ${count} transaction${count === 1 ? "" : "s"}`;
});

const todayInUserTimezone = computed(() => {
  try {
    const tz = userSettings.value?.timezone;
    return (tz ? dayjs().tz(tz) : dayjs()).startOf("day").toDate();
  } catch {
    return dayjs().startOf("day").toDate();
  }
});

const rules = {
  account: { name: { required } },
  txn_date: { required },
  description: {},
  rows: {
    $each: {
      direction: { required },
      amount: {
        required,
        decimalValid,
        decimalMin: decimalMin("0.01"),
        decimalMax: decimalMax(1_000_000_000),
      },
      category: { name: {} },
      description: {},
    },
  },
};

const { r$ } = useRegle(record, rules);

onMounted(async () => {
  await getSettings();

  const defaultChecking = accounts.value.find(
    (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
  );
  if (defaultChecking) {
    record.value.account = defaultChecking;
    defaultPreSelected.value = true;
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

function newRow(): BulkTransactionRow {
  return {
    direction: selectedParentCategory.value?.classification ?? "expense",
    amount: null,
    category: null,
    description: null,
    idempotency_key: crypto.randomUUID(),
  };
}

function addRow() {
  if (record.value.rows.length < MAX_ROWS) record.value.rows.push(newRow());
}

function removeRow(index: number) {
  if (record.value.rows.length > 1) record.value.rows.splice(index, 1);
}

function syncRowDirections() {
  const direction = selectedParentCategory.value?.classification;
  if (!direction) return;
  record.value.rows.forEach((row) => {
    if (row.direction === direction) return;
    row.direction = direction;
    row.category = null;
  });
}

watch(sharedDirection, (on) => {
  if (on) syncRowDirections();
});

const searchAccount = (event: { query: string }) => {
  filteredAccounts.value = searchHelper.filterByQuery(
    accounts.value,
    event.query,
    (acc) => [acc.name],
    { sort: true },
  );
};

async function manageRecord() {
  if (submitting.value) return;

  submitting.value = true;
  try {
    const { valid } = await r$.$validate();
    if (!valid) return;

    const txn_date = dateHelper.mergeDateWithCurrentTime(
      dayjs(record.value.txn_date).format("YYYY-MM-DD"),
      userSettings.value?.timezone || "UTC",
    );

    const response = await transactionStore.bulkInsertTransactions({
      account_id: record.value.account!.id!,
      txn_date,
      rows: record.value.rows.map((row) => ({
        direction: row.direction,
        amount: row.amount!,
        category_id: row.category?.id,
        description: sharedDescription.value
          ? record.value.description
          : row.description,
        idempotency_key: row.idempotency_key,
      })),
    });

    if (response?.code === 208) {
      toastStore.infoResponseToast(response);
    } else {
      toastStore.successResponseToast(response);
    }
    r$.$reset();
    emit("completeTxOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <BaseForm
    class="flex flex-col gap-4 p-1"
    :disabled="submitting"
    @submit="manageRecord"
  >
    <div v-if="sharedDirection" class="flex flex-row w-full justify-center">
      <SelectButton
        v-model="selectedParentCategory"
        style="font-size: 0.875rem"
        size="small"
        :options="parentCategories"
        option-label="display_name"
        :allow-empty="false"
        @update:model-value="syncRowDirections"
      />
    </div>

    <span
      v-if="defaultPreSelected"
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      Default checking account pre-selected.
    </span>

    <div class="flex flex-col sm:flex-row w-full gap-4">
      <div class="flex flex-col flex-1 gap-1 min-w-0">
        <ValidationError
          :is-required="true"
          :message="r$.account.name.$errors[0]"
        >
          <label>Account</label>
        </ValidationError>
        <AutoComplete
          v-model="record.account"
          size="small"
          :suggestions="filteredAccounts"
          option-label="name"
          overlay-class="w-0"
          force-selection
          placeholder="Select account"
          dropdown
          @complete="searchAccount"
          @update:model-value="defaultPreSelected = false"
        />
        <AvailableBalance v-if="hasExpenseRow" :account="record.account" />
      </div>
      <div class="flex flex-col flex-1 gap-1 min-w-0">
        <ValidationError :is-required="true" :message="r$.txn_date.$errors[0]">
          <label>Date</label>
        </ValidationError>
        <DatePicker
          v-model="record.txn_date"
          date-format="dd/mm/yy"
          show-icon
          fluid
          icon-display="input"
          size="small"
          :max-date="todayInUserTimezone"
        />
      </div>
    </div>

    <div v-if="sharedDescription" class="flex flex-col gap-1 w-full">
      <ValidationError
        :is-required="false"
        :message="r$.description.$errors[0]"
      >
        <label>Description</label>
      </ValidationError>
      <InputText
        v-model="record.description"
        size="small"
        placeholder="Describe transactions"
      />
    </div>

    <div class="flex flex-col gap-3">
      <TransactionBulkRow
        v-for="(row, i) in record.rows"
        :key="i"
        v-model="record.rows[i]"
        :directions="parentCategories"
        :categories="categoriesByDirection[row.direction] ?? []"
        :errors="{
          direction: r$.rows.$each[i]?.direction.$errors[0],
          amount: r$.rows.$each[i]?.amount.$errors[0],
          category: r$.rows.$each[i]?.category.name.$errors[0],
          description: r$.rows.$each[i]?.description.$errors[0],
        }"
        :show-direction="!sharedDirection"
        :show-description="!sharedDescription"
        :removable="record.rows.length > 1"
        @remove="removeRow(i)"
      />
    </div>

    <div class="flex flex-row w-full items-center justify-between gap-2">
      <Button
        label="Add row"
        icon="pi pi-plus"
        severity="secondary"
        text
        size="small"
        :disabled="record.rows.length >= MAX_ROWS"
        @click="addRow"
      />
      <span class="text-sm" style="color: var(--text-secondary)">
        {{ sharedDirection ? "Total" : "Net" }}:
        {{ vueHelper.displayAsCurrency(total) }}
      </span>
    </div>

    <Button
      class="main-button"
      :label="submitLabel"
      :disabled="submitting"
      :loading="submitting"
      @click="manageRecord"
    />
  </BaseForm>
</template>

<style scoped></style>
