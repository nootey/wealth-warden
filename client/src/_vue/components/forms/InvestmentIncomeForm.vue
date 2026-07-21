<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { required, requiredIf } from "@regle/rules";
import { useRegle } from "@regle/core";
import ValidationError from "../validation/ValidationError.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useInvestmentStore } from "../../../services/stores/investment_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import dayjs from "dayjs";
import type {
  IncomeType,
  InvestmentType,
} from "../../../models/investment_models.ts";
import type { UserSettings } from "../../../models/settings_models.ts";

const props = defineProps<{
  assetId: number;
  assetCurrency: string;
  investmentType: InvestmentType;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
}>();

const toastStore = useToastStore();
const investmentStore = useInvestmentStore();
const settingsStore = useSettingsStore();

const loading = ref(false);
const userSettings = ref<UserSettings>();

const isStaking = computed(() => props.investmentType === "crypto");

const record = ref({
  income_type: (props.investmentType === "crypto"
    ? "staking_reward"
    : "dividend") as IncomeType,
  txn_date: dayjs().toDate(),
  quantity: "",
  amount: "",
  tax_withheld: "",
  notes: "",
});

const quantityRef = computed({
  get: () => record.value.quantity,
  set: (v) => (record.value.quantity = v),
});
const { number: quantityNumber } = currencyHelper.useMoneyField(quantityRef, 8);

const amountRef = computed({
  get: () => record.value.amount,
  set: (v) => (record.value.amount = v),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 4);

const taxRef = computed({
  get: () => record.value.tax_withheld,
  set: (v) => (record.value.tax_withheld = v),
});
const { number: taxNumber } = currencyHelper.useMoneyField(taxRef, 4);

const rules = {
  txn_date: { required },
  quantity: {
    requiredIf: requiredIf(() => isStaking.value),
  },
  amount: {
    requiredIf: requiredIf(() => !isStaking.value),
  },
  notes: {},
};

const { r$ } = useRegle(record, rules);

onMounted(async () => {
  try {
    const res = await settingsStore.getUserSettings();
    userSettings.value = res.data;
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
});

async function submit(): Promise<void> {
  const { valid } = await r$.$validate();
  if (!valid) return;

  loading.value = true;

  const txn_date = dateHelper.mergeDateWithCurrentTime(
    dayjs(record.value.txn_date).format("YYYY-MM-DD"),
    userSettings.value?.timezone || "UTC",
  );

  const payload: Record<string, unknown> = {
    asset_id: props.assetId,
    income_type: record.value.income_type,
    txn_date,
    currency: props.assetCurrency,
    notes: record.value.notes || null,
  };

  if (isStaking.value) {
    payload.quantity = record.value.quantity;
  } else {
    payload.amount = record.value.amount;
    if (record.value.tax_withheld) {
      payload.tax_withheld = record.value.tax_withheld;
    }
  }

  try {
    const response = await investmentStore.createIncome(payload);
    r$.$reset();
    toastStore.successResponseToast(response);
    emit("completeOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div
    class="flex flex-col gap-4 p-4 rounded-xl"
    style="
      background-color: var(--background-secondary);
      border: 1px solid var(--border-color);
    "
  >
    <div class="flex flex-row w-full gap-4">
      <div v-if="isStaking" class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="true" :message="r$.quantity.$errors[0]">
          <label>Quantity received</label>
        </ValidationError>
        <InputNumber
          v-model="quantityNumber"
          size="small"
          locale="de-DE"
          :min-fraction-digits="2"
          :max-fraction-digits="8"
          placeholder="0"
        />
      </div>
      <div v-if="!isStaking" class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="true" :message="r$.amount.$errors[0]">
          <label>Amount received</label>
        </ValidationError>
        <InputNumber
          v-model="amountNumber"
          size="small"
          locale="de-DE"
          :min-fraction-digits="2"
          :max-fraction-digits="4"
          placeholder="0,00"
        />
      </div>
    </div>

    <div class="flex flex-row w-full gap-4">
      <div class="flex flex-col gap-1 w-6/12">
        <ValidationError :is-required="true" :message="r$.txn_date.$errors[0]">
          <label>Date</label>
        </ValidationError>
        <DatePicker
          v-model="record.txn_date"
          size="small"
          date-format="dd/mm/yy"
          show-icon
        />
      </div>
      <div v-if="!isStaking" class="flex flex-col gap-1 w-6/12">
        <ValidationError :is-required="false" :message="undefined">
          <label>Tax withheld</label>
        </ValidationError>
        <InputNumber
          v-model="taxNumber"
          size="small"
          locale="de-DE"
          :min-fraction-digits="2"
          :max-fraction-digits="4"
          placeholder="0,00"
        />
      </div>
      <div v-if="isStaking" class="flex flex-col gap-1 w-6/12">
        <ValidationError :is-required="false" :message="r$.notes.$errors[0]">
          <label>Notes</label>
        </ValidationError>
        <InputText
          v-model="record.notes"
          size="small"
          placeholder="Describe trade ..."
        />
      </div>
    </div>

    <div v-if="!isStaking" class="flex flex-row w-full gap-4">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="false" :message="r$.notes.$errors[0]">
          <label>Notes</label>
        </ValidationError>
        <InputText
          v-model="record.notes"
          size="small"
          placeholder="Describe trade ..."
        />
      </div>
    </div>

    <Button
      label="Save income"
      class="main-button"
      style="height: 38px"
      :disabled="loading"
      @click="submit"
    />
  </div>
</template>

<style scoped></style>
