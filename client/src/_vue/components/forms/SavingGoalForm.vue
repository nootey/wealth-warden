<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useSavingsStore } from "../../../services/stores/savings_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { numeric, required } from "@vuelidate/validators";
import { decimalMin, decimalValid } from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import { useConfirm } from "primevue/useconfirm";
import type { Account } from "../../../models/account_models.ts";
import type {
  SavingGoalReq,
  SavingGoalUpdateReq,
  SavingGoalStatus,
} from "../../../models/savings_models.ts";
import Decimal from "decimal.js";
import dateHelper from "../../../utils/date_helper.ts";
import AuditTrail from "../base/AuditTrail.vue";

interface GoalFormData {
  account_id: number | null;
  name: string;
  target_amount: string | null;
  initial_amount: string | null;
  target_date: Date | null;
  status: SavingGoalStatus;
  priority: number;
  monthly_allocation: string | null;
}

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const toastStore = useToastStore();
const accountStore = useAccountStore();
const savingsStore = useSavingsStore();
const settingsStore = useSettingsStore();
const confirm = useConfirm();

const initializing = ref(false);
const submitting = ref(false);
const accounts = ref<Account[]>([]);

const record = ref<GoalFormData>(initData());

const targetAmountRef = computed({
  get: () => record.value.target_amount,
  set: (v) => (record.value.target_amount = v),
});
const { number: targetAmountNumber } = currencyHelper.useMoneyField(
  targetAmountRef,
  2,
);

const monthlyAllocationRef = computed({
  get: () => record.value.monthly_allocation,
  set: (v) => (record.value.monthly_allocation = v),
});
const { number: monthlyAllocationNumber } = currencyHelper.useMoneyField(
  monthlyAllocationRef,
  2,
);

const initialAmountRef = computed({
  get: () => record.value.initial_amount,
  set: (v) => (record.value.initial_amount = v),
});
const { number: initialAmountNumber } = currencyHelper.useMoneyField(
  initialAmountRef,
  2,
);

const statusOptions: { label: string; value: SavingGoalStatus }[] = [
  { label: "Active", value: "active" },
  { label: "Paused", value: "paused" },
  { label: "Completed", value: "completed" },
  { label: "Archived", value: "archived" },
];

const rules = computed(() => ({
  record: {
    account_id: { required, $autoDirty: true },
    name: { required, $autoDirty: true },
    target_date: { $autoDirty: true },
    target_amount: {
      required,
      decimalValid,
      decimalMin: decimalMin("0.01"),
      $autoDirty: true,
    },
    initial_amount: {
      $autoDirty: true,
    },
    monthly_allocation: {
      $autoDirty: true,
    },
    priority: {
      numeric,
      $autoDirty: true,
    },
    status: {
      required,
      $autoDirty: true,
    },
  },
}));

const v$ = useVuelidate(rules, { record });

onMounted(async () => {
  initializing.value = true;
  try {
    const all = await accountStore.getAccountsBySubtype("savings");
    accounts.value = (all as Account[]).filter((a) => a.is_active);

    if (props.mode === "update" && props.recordId) {
      await loadRecord(props.recordId);
    }
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    initializing.value = false;
  }
});

function initData(): GoalFormData {
  return {
    account_id: null,
    name: "",
    target_amount: null,
    initial_amount: null,
    target_date: null,
    status: "active",
    priority: 0,
    monthly_allocation: null,
  };
}

async function loadRecord(id: number) {
  const goal = await savingsStore.fetchGoalByID(id);
  record.value = {
    initial_amount: null,
    account_id: goal.account_id,
    name: goal.name,
    target_amount: goal.target_amount,
    target_date: goal.target_date ? new Date(goal.target_date) : null,
    status: goal.status,
    priority: goal.priority,
    monthly_allocation: goal.monthly_allocation ?? null,
  };
}

async function isRecordValid() {
  return await v$.value.record.$validate();
}

async function manageRecord() {
  if (submitting.value) return;
  submitting.value = true;

  if (!(await isRecordValid())) {
    submitting.value = false;
    return;
  }

  const targetDate = record.value.target_date
    ? dateHelper.formatDate(record.value.target_date)
    : null;

  try {
    let response = null;

    if (props.mode === "create") {
      const req: SavingGoalReq = {
        account_id: record.value.account_id!,
        name: record.value.name,
        target_amount: new Decimal(record.value.target_amount!).toFixed(2),
        initial_amount: record.value.initial_amount
          ? new Decimal(record.value.initial_amount).toFixed(2)
          : null,
        target_date: targetDate,
        priority: record.value.priority,
        monthly_allocation: record.value.monthly_allocation
          ? new Decimal(record.value.monthly_allocation).toFixed(2)
          : null,
      };
      response = await savingsStore.insertGoal(req);
    } else {
      const req: SavingGoalUpdateReq = {
        name: record.value.name,
        target_amount: new Decimal(record.value.target_amount!).toFixed(2),
        target_date: targetDate,
        status: record.value.status,
        priority: record.value.priority,
        monthly_allocation: record.value.monthly_allocation
          ? new Decimal(record.value.monthly_allocation).toFixed(2)
          : null,
      };
      response = await savingsStore.updateGoal(props.recordId!, req);
    }

    v$.value.record.$reset();
    toastStore.successResponseToast(response);
    emit("completeOperation");
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    submitting.value = false;
  }
}

function confirmDelete() {
  confirm.require({
    message: "Delete this goal and all its contributions?",
    header: "Confirm delete",
    icon: "pi pi-exclamation-triangle",
    rejectProps: { label: "Cancel", severity: "secondary", outlined: true },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: async () => {
      submitting.value = true;
      try {
        const res = await savingsStore.deleteGoal(props.recordId!);
        toastStore.successResponseToast(res);
        emit("completeDelete");
      } catch (err) {
        toastStore.errorResponseToast(err);
      } finally {
        submitting.value = false;
      }
    },
  });
}
</script>

<template>
  <div v-if="!initializing" class="flex flex-column gap-3 p-1">
    <div v-if="mode === 'create'" class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.account_id.$errors[0]?.$message"
      >
        <label>Account</label>
      </ValidationError>
      <Select
        v-model="record.account_id"
        :options="accounts"
        option-label="name"
        option-value="id"
        placeholder="Select savings account"
        size="small"
        filter
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.name.$errors[0]?.$message"
      >
        <label>Name</label>
      </ValidationError>
      <InputText
        v-model="record.name"
        placeholder="e.g. Emergency fund"
        size="small"
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.target_amount.$errors[0]?.$message"
      >
        <label>Target amount</label>
      </ValidationError>
      <InputNumber
        v-model="targetAmountNumber"
        size="small"
        mode="currency"
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
    </div>

    <div v-if="mode === 'create'" class="flex flex-column gap-1">
      <ValidationError
        :is-required="false"
        :message="v$.record.initial_amount.$errors[0]?.$message"
      >
        <label>Initial amount</label>
      </ValidationError>
      <InputNumber
        v-model="initialAmountNumber"
        size="small"
        mode="currency"
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="false"
        :message="v$.record.target_date.$errors[0]?.$message"
      >
        <label>Target date</label>
      </ValidationError>
      <DatePicker
        v-model="record.target_date"
        placeholder="Pick a date"
        date-format="dd/mm/yy"
        size="small"
        show-button-bar
        fluid
        icon-display="input"
        show-icon
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="false"
        :message="v$.record.priority.$errors[0]?.$message"
      >
        <label>Priority</label>
      </ValidationError>
      <InputNumber
        v-model="record.priority"
        size="small"
        :min="0"
        :max="999"
        placeholder="0"
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="false"
        :message="v$.record.monthly_allocation.$errors[0]?.$message"
      >
        <label>Monthly allocation</label>
      </ValidationError>
      <InputNumber
        v-model="monthlyAllocationNumber"
        size="small"
        mode="currency"
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
    </div>

    <div v-if="mode === 'update'" class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.status.$errors[0]?.$message"
      >
        <label>Status</label>
      </ValidationError>
      <Select
        v-model="record.status"
        :options="statusOptions"
        option-label="label"
        option-value="value"
        size="small"
      />
    </div>

    <div class="flex flex-column gap-2 w-full">
      <div class="flex flex-row w-full">
        <Button
          class="main-button w-full"
          :label="mode === 'create' ? 'Add goal' : 'Update goal'"
          :disabled="submitting"
          :loading="submitting"
          style="height: 42px"
          @click="manageRecord"
        />
      </div>
      <div class="flex flex-row w-full">
        <Button
          v-if="mode === 'update'"
          class="delete-button w-full"
          label="Delete goal"
          style="height: 42px"
          :disabled="submitting"
          @click="confirmDelete"
        />
      </div>
    </div>

    <div v-if="mode == 'update'" class="flex flex-row gap-2 w-full">
      <AuditTrail
        :record-id="props.recordId!"
        :events="['create', 'update', 'delete']"
        category="saving_goal"
      />
    </div>
  </div>

  <ShowLoading v-else :num-fields="5" />
</template>

<style scoped></style>
