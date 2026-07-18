<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { required, minValue } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import ValidationError from "./validation/ValidationError.vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useInvestmentStore } from "../../services/stores/investment_store.ts";
import { useSettingsStore } from "../../services/stores/settings_store.ts";
import currencyHelper from "../../utils/currency_helper.ts";
import vueHelper from "../../utils/vue_helper.ts";
import {
  decimalMax,
  decimalMin,
  decimalValid,
} from "../../validators/currency.ts";
import type {
  InvestmentTaxBracket,
  InvestmentTaxSettings,
  InvestmentType,
} from "../../models/investment_models.ts";

const toastStore = useToastStore();
const investmentStore = useInvestmentStore();
const settingsStore = useSettingsStore();

const loading = ref(false);
const brackets = ref<InvestmentTaxBracket[]>([]);
const settings = ref<InvestmentTaxSettings>({ loss_offsetting_enabled: false });

const addBracketModal = ref(false);
const addBracketType = ref<InvestmentType>("stock");

const INVESTMENT_TYPES: InvestmentType[] = ["stock", "etf", "crypto"];
const TYPE_LABELS: Record<InvestmentType, string> = {
  stock: "Stock",
  etf: "ETF",
  crypto: "Crypto",
};

const bracketsByType = computed(() => {
  const result: Record<InvestmentType, InvestmentTaxBracket[]> = {
    stock: [],
    etf: [],
    crypto: [],
  };
  for (const b of brackets.value) {
    result[b.investment_type].push(b);
  }
  return result;
});

const form = ref({
  min_days_held: null as number | null,
  to_days: null as number | null,
  taxable_percent: "" as string,
  label: "",
});

const taxablePercentRef = computed({
  get: () => form.value.taxable_percent,
  set: (v) => (form.value.taxable_percent = v),
});
const { number: taxablePercentNumber } = currencyHelper.useMoneyField(
  taxablePercentRef,
  2,
);

const rules = {
  form: {
    min_days_held: { required, minValue: minValue(0), $autoDirty: true },
    taxable_percent: {
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(100),
      $autoDirty: true,
    },
  },
};

const v$ = useVuelidate(rules, { form });

onMounted(async () => {
  await Promise.all([loadBrackets(), loadSettings()]);
});

async function loadBrackets(): Promise<void> {
  try {
    brackets.value = await investmentStore.getTaxBrackets();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function loadSettings(): Promise<void> {
  try {
    settings.value = await investmentStore.getTaxSettings();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

function openAddBracket(type: InvestmentType): void {
  addBracketType.value = type;
  form.value = {
    min_days_held: null,
    to_days: null,
    taxable_percent: "",
    label: "",
  };
  v$.value.form.$reset();
  addBracketModal.value = true;
}

async function submitBracket(): Promise<void> {
  const valid = await v$.value.$validate();
  if (!valid) return;

  loading.value = true;
  try {
    const response = await investmentStore.createTaxBracket({
      investment_type: addBracketType.value,
      min_days_held: form.value.min_days_held,
      to_days: form.value.to_days ?? null,
      taxable_percent: form.value.taxable_percent,
      label: form.value.label || null,
    });
    toastStore.successResponseToast(response);
    addBracketModal.value = false;
    await loadBrackets();
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

function copySourceOptions(
  type: InvestmentType,
): { label: string; value: InvestmentType }[] {
  return INVESTMENT_TYPES.filter(
    (t) => t !== type && bracketsByType.value[t].length > 0,
  ).map((t) => ({ label: TYPE_LABELS[t], value: t }));
}

async function copyBrackets(
  fromType: InvestmentType,
  toType: InvestmentType,
): Promise<void> {
  loading.value = true;
  try {
    const response = await investmentStore.copyTaxBrackets(fromType, toType);
    toastStore.successResponseToast(response);
    await loadBrackets();
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function deleteBracket(id: number): Promise<void> {
  try {
    const response = await investmentStore.deleteTaxBracket(id);
    toastStore.successResponseToast(response);
    await loadBrackets();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function saveSettings(): Promise<void> {
  loading.value = true;
  try {
    const response = await investmentStore.saveTaxSettings({
      loss_offsetting_enabled: settings.value.loss_offsetting_enabled,
    });
    toastStore.successResponseToast(response);
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <Dialog
    v-model:visible="addBracketModal"
    :modal="true"
    :style="{ width: '400px' }"
    :breakpoints="{ '451px': '90vw' }"
    class="rounded-dialog"
    :header="`Add ${TYPE_LABELS[addBracketType]} Bracket`"
  >
    <div class="flex flex-col gap-4 p-1">
      <div class="flex flex-col gap-4">
        <div class="flex flex-row w-full gap-2 items-center">
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="v$.form.min_days_held.$errors[0]?.$message"
            >
              <label>Min days held</label>
            </ValidationError>
            <InputNumber
              v-model="form.min_days_held"
              size="small"
              :locale="
                vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)
              "
              :min="0"
              :max-fraction-digits="0"
              placeholder="0"
              fluid
            />
          </div>
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError :is-required="false" :message="undefined">
              <label>To days</label>
            </ValidationError>
            <InputNumber
              v-model="form.to_days"
              size="small"
              :locale="
                vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)
              "
              :min="0"
              :max-fraction-digits="0"
              placeholder="∞"
              fluid
            />
          </div>
        </div>

        <div class="flex flex-row w-full gap-2 items-center">
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="v$.form.taxable_percent.$errors[0]?.$message"
            >
              <label>Taxable %</label>
            </ValidationError>
            <InputNumber
              v-model="taxablePercentNumber"
              size="small"
              :locale="
                vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)
              "
              :min="0"
              :max="100"
              :min-fraction-digits="2"
              :max-fraction-digits="2"
              placeholder="0,00"
              fluid
            />
          </div>
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError :is-required="false" :message="undefined">
              <label>Label</label>
            </ValidationError>
            <InputText
              v-model="form.label"
              size="small"
              placeholder="e.g. Short-term"
            />
          </div>
        </div>
      </div>

      <Button
        label="Save bracket"
        class="main-button"
        style="height: 38px"
        :disabled="loading"
        @click="submitBracket"
      />
    </div>
  </Dialog>

  <div class="flex flex-col gap-4">
    <div
      class="flex flex-row items-center justify-between p-4 rounded-xl gap-4"
      style="
        background-color: var(--background-secondary);
        border: 1px solid var(--border-color);
      "
    >
      <div class="flex flex-col w-full gap-4">
        <div class="flex flex-row items-center text-center gap-2 w-full">
          <h4>Settings</h4>
        </div>

        <div
          class="flex flex-row items-center text-center gap-2 p-4 w-full rounded-lg"
          style="border: 1px solid var(--border-color)"
        >
          <span class="font-semibold text-sm">Loss offsetting</span>
          <span> - </span>
          <span class="text-sm" style="color: var(--text-secondary)">
            When enabled, losses across open lots offset gains before
            calculating tax.
          </span>
          <ToggleSwitch
            v-model="settings.loss_offsetting_enabled"
            class="ml-auto"
          />
        </div>

        <div class="flex flex-row items-center">
          <Button
            label="Save"
            class="main-button ml-auto"
            style="height: 32px"
            :disabled="loading"
            @click="saveSettings"
          />
        </div>
      </div>
    </div>

    <div
      v-for="type in INVESTMENT_TYPES"
      :key="type"
      class="flex flex-col p-4 gap-4 rounded-xl"
      style="
        background-color: var(--background-secondary);
        border: 1px solid var(--border-color);
      "
    >
      <div class="flex flex-row justify-between items-center gap-2">
        <span class="font-bold">{{ TYPE_LABELS[type] }}</span>
        <div class="flex flex-row gap-2 items-center">
          <Select
            v-if="
              bracketsByType[type].length === 0 &&
              copySourceOptions(type).length > 0
            "
            :options="copySourceOptions(type)"
            option-label="label"
            option-value="value"
            size="small"
            placeholder="Copy from..."
            :disabled="loading"
            @change="(e) => copyBrackets(e.value, type)"
          />
          <Button
            class="main-button"
            style="height: 30px"
            @click="openAddBracket(type)"
          >
            <div class="flex flex-row gap-1 items-center">
              <i class="pi pi-plus" />
              <span>Add bracket</span>
            </div>
          </Button>
        </div>
      </div>

      <span
        v-if="bracketsByType[type].length === 0"
        class="text-sm"
        style="color: var(--text-secondary)"
      >
        No brackets configured - tax info will not be shown for
        {{ TYPE_LABELS[type].toLowerCase() }}.
      </span>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="b in bracketsByType[type]"
          :key="b.id"
          class="flex flex-row items-center justify-between p-2 rounded-lg gap-2"
          style="border: 1px solid var(--border-color)"
        >
          <div class="flex flex-row gap-4 items-center flex-wrap">
            <span class="font-semibold text-sm">
              {{ b.min_days_held }} – {{ b.to_days ?? "∞" }} days
            </span>
            <span class="text-sm">{{ b.taxable_percent }}% taxable</span>
            <span
              v-if="b.label"
              class="text-sm"
              style="color: var(--text-secondary)"
            >
              {{ b.label }}
            </span>
          </div>
          <Button
            icon="pi pi-trash"
            severity="danger"
            text
            size="small"
            @click="deleteBracket(b.id)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
