<script setup lang="ts">
import { computed, ref } from "vue";
import { required } from "@vuelidate/validators";
import { decimalNonZero, decimalValid } from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSavingsStore } from "../../../services/stores/savings_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import Decimal from "decimal.js";
import type { SavingContributionReq } from "../../../models/savings_models.ts";

const props = defineProps<{
  goalId: number;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
}>();

const toastStore = useToastStore();
const savingsStore = useSavingsStore();
const settingsStore = useSettingsStore();

const submitting = ref(false);

const record = ref({
  amount: null as string | null,
  month: new Date() as Date | null,
  note: "" as string,
});

const amountRef = computed({
  get: () => record.value.amount,
  set: (v) => (record.value.amount = v),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const rules = computed(() => ({
  record: {
    amount: { required, decimalValid, decimalNonZero, $autoDirty: true },
    month: { required, $autoDirty: true },
  },
}));

const v$ = useVuelidate(rules, { record });

async function manageRecord() {
  if (submitting.value) return;
  submitting.value = true;

  if (!(await v$.value.record.$validate())) {
    submitting.value = false;
    return;
  }

  try {
    const req: SavingContributionReq = {
      amount: new Decimal(record.value.amount!).toFixed(2),
      month: dateHelper.formatDate(record.value.month!),
      note: record.value.note || null,
    };
    const response = await savingsStore.insertContribution(props.goalId, req);
    toastStore.successResponseToast(response);
    v$.value.record.$reset();
    emit("completeOperation");
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="flex flex-column gap-3 p-1">
    <div
      class="flex flex-column gap-2 p-3 border-round-xl text-sm"
      style="
        background: var(--background-secondary);
        border: 1px solid var(--border-color);
        color: var(--text-secondary);
      "
    >
      <div class="flex flex-row gap-2 align-items-center">
        <i class="pi pi-info-circle" style="flex-shrink: 0" />
        <div class="flex flex-column gap-1 text-xs">
          <span
            >To take some allocation off this goal, add a negative
            contribution.</span
          >
        </div>
      </div>
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.amount.$errors[0]?.$message"
      >
        <label>Amount</label>
      </ValidationError>
      <InputNumber
        v-model="amountNumber"
        size="small"
        mode="currency"
        fluid
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError
        :is-required="true"
        :message="v$.record.month.$errors[0]?.$message"
      >
        <label>Month</label>
      </ValidationError>
      <DatePicker
        v-model="record.month"
        size="small"
        fluid
        view="month"
        date-format="mm/yy"
        placeholder="Month"
      />
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError :is-required="false" :message="undefined">
        <label>Note</label>
      </ValidationError>
      <InputText
        v-model="record.note"
        placeholder="e.g. Bonus payout"
        size="small"
      />
    </div>

    <div class="flex flex-row w-full">
      <Button
        class="main-button w-full"
        label="Add contribution"
        :disabled="submitting"
        :loading="submitting"
        style="height: 42px"
        @click="manageRecord"
      />
    </div>
  </div>
</template>

<style scoped></style>
