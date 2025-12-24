<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import type {
  InvestmentHolding, InvestmentTransaction,
} from "../../../models/investment_models.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import { required } from "@vuelidate/validators";
import {
  decimalMax,
  decimalMin,
  decimalValid,
} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import dayjs from "dayjs";
import dateHelper from "../../../utils/date_helper.ts";
import {useInvestmentStore} from "../../../services/stores/investment_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const apiPrefix = "investments/transactions";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const investmentStore = useInvestmentStore();

const loading = ref(false);
const isReadOnly = computed(
  () => props.mode === "update",
);

const holdings = ref<InvestmentHolding[]>([]);
const record = ref<InvestmentTransaction>(initData());
const filteredHoldings = ref<InvestmentHolding[]>([]);

const transactionTypes = ref<string[]>(["Buy", "Sell"]);

const selectedTransactionType = ref<string>(
  transactionTypes.value.find((i) => i === "Buy") ?? "Sell",
);

const availableCurrencies = ref<string[]>(["USD", "EUR", "GBP"]);

const quantityRef = computed({
  get: () => record.value.quantity,
  set: (v) => (record.value.quantity = v),
});
const { number: quantityNumber } = currencyHelper.useMoneyField(quantityRef, 2);

const feeRef = computed({
  get: () => record.value.fee,
  set: (v) => (record.value.fee = v),
});
const { number: feeNumber } = currencyHelper.useMoneyField(feeRef, 2);

const pricePerUnitRef = computed({
  get: () => record.value.price_per_unit,
  set: (v) => (record.value.price_per_unit = v),
});
const { number: pricePerUnitNumber } = currencyHelper.useMoneyField(pricePerUnitRef, 2);

const rules = {
  record: {
    holding: {
      required,
      $autoDirty: true,
    },
    txn_date: {
      required,
      $autoDirty: true,
    },
    transaction_type: {
      required,
      $autoDirty: true,
    },
    quantity: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
      $autoDirty: true,
    },
    fee: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
      $autoDirty: true,
    },
    price_per_unit: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
      $autoDirty: true,
    },
    currency: {
      required,
      $autoDirty: true,
    },
    description: {
      $autoDirty: true,
    },
  }
};

const v$ = useVuelidate(rules, { record });

onMounted(async () => {

  holdings.value = await investmentStore.getAllHoldings();

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

function initData(): InvestmentTransaction {
  return {
    holding: null,
    txn_date: dayjs().toDate(),
    transaction_type: "buy",
    quantity: "",
    fee: "0",
    price_per_unit: "",
    currency: "USD",
    description: "",
  };
}

function getCurrencyPlaceholder(currency: string) {
  if(record.value.holding?.investment_type === 'crypto')
    return '0'

  const symbols: Record<string, string> = {
    USD: '$',
    EUR: '€',
    GBP: '£'
  };
  return `0,00 ${symbols[currency] || currency}`;
}

const searchHolding = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredHoldings.value = [...holdings.value];
    } else {
      filteredHoldings.value = holdings.value.filter((record) => {
        return record.ticker.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
};

async function isRecordValid() {
  const isValid = await v$.value.$validate();
  if (!isValid) return false;
  return true;
}

async function loadRecord(id: number) {
  try {
    loading.value = true;
    const data = await sharedStore.getRecordByID(apiPrefix, id, {
      deleted: true,
    });

    record.value = {
      ...initData(),
      ...data,
      txn_date: data.txn_date
        ? dayjs(data.txn_date).toDate()
        : dayjs().toDate(),
    };

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function manageRecord() {
  if (!(await isRecordValid())) return;
  if (!record.value.holding) return;

  const txn_date = dateHelper.mergeDateWithCurrentTime(
    dayjs(record.value.txn_date).format("YYYY-MM-DD"),
  );

  const recordData = {
    holding_id: record.value.holding.id,
    transaction_type: selectedTransactionType.value.toLowerCase(),
    txn_date: txn_date,
    quantity: record.value.quantity,
    price_per_unit: record.value.price_per_unit,
    fee: record.value.fee,
    currency: record.value.currency,
    description: record.value.description,
  };

  try {
    let response = null;

    switch (props.mode) {
      case "create":
        response = await sharedStore.createRecord(apiPrefix, recordData);
        break;
      case "update":
        response = await sharedStore.updateRecord(
          apiPrefix,
          record.value.id!,
          recordData,
        );
        break;
      default:
        emit("completeOperation");
        break;
    }

    v$.value.record.$reset();
    toastStore.successResponseToast(response);
    emit("completeOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}
</script>

<template>

  <div v-if="!loading" class="flex flex-column gap-3 p-1">

    <div v-if="!isReadOnly" class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column">
        <SelectButton
          v-model="selectedTransactionType"
          style="font-size: 0.875rem"
          size="small"
          :options="transactionTypes"
          :allow-empty="false"
          :readonly="isReadOnly"
          :disabled="isReadOnly"
        />
      </div>
    </div>

    <span v-if="isReadOnly" class="text-sm" style="color: var(--text-secondary)">
      This is a read only view.
      Due to the complexity of re-calculating the financial impact of the transaction, most fields can not be updated.
    </span>

    <span v-if="isReadOnly" class="text-sm" style="color: var(--text-secondary)">
      If you wish to make changes, delete the transaction and create a new one.
    </span>

    <div v-if="mode==='update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>Transaction type</label>
        <span style="color: var(--text-secondary)">{{record.transaction_type}}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>USD exchange rate</label>
        <span style="color: var(--text-secondary)">{{record.exchange_rate_to_usd}}</span>
      </div>
    </div>

    <div v-if="mode==='update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>Value at buy</label>
        <span style="color: var(--text-secondary)">{{vueHelper.displayAsCurrency(record.value_at_buy!)}}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>Current value</label>
        <span style="color: var(--text-secondary)">{{vueHelper.displayAsCurrency(record.current_value!)}}</span>
      </div>
    </div>

    <div v-if="mode==='update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>P&L Raw</label>
        <span style="color: var(--text-secondary)">{{vueHelper.displayAsCurrency(record.profit_loss!)}}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>P&L Percentage</label>
        <span style="color: var(--text-secondary)">{{vueHelper.displayAsPercentage(record.profit_loss_percent!)}}</span>
      </div>
    </div>

    <div class="flex flex-column gap-3">

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.holding.$errors[0]?.$message"
          >
            <label>Holding</label>
          </ValidationError>
          <AutoComplete
            v-model="record.holding"
            size="small"
            :suggestions="filteredHoldings"
            option-label="name"
            data-key="id"
            force-selection
            placeholder="Select holding"
            dropdown
            @complete="searchHolding"
            :readonly="isReadOnly"
            :disabled="isReadOnly"
          >
            <template #option="slotProps">
              <div class="flex align-items-center gap-2">
                <span class="font-semibold">{{ slotProps.option.name }}</span>
                <span class="text-color-secondary">{{ slotProps.option.ticker }}</span>
              </div>
            </template>

            <template #chip="slotProps">
              <div class="flex align-items-center gap-2">
                <span class="font-semibold">{{ slotProps.value.name }}</span>
                <span class="text-color-secondary">{{ slotProps.value.ticker }}</span>
              </div>
            </template>
          </AutoComplete>
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
            :readonly="isReadOnly"
            :disabled="isReadOnly"
          />
        </div>
      </div>

      <div class="flex flex-row w-full gap-3">
        <div class="flex flex-column gap-1 w-6">
          <ValidationError
            :is-required="true"
            :message="v$.record.quantity.$errors[0]?.$message"
          >
            <label>Quantity</label>
          </ValidationError>
          <InputNumber
            v-model="quantityNumber"
            size="small"
            locale="de-DE"
            :min-fraction-digits="2"
            :max-fraction-digits="6"
            placeholder="0,00"
            :readonly="isReadOnly"
            :disabled="isReadOnly"
            fluid
          />
        </div>
        <div class="flex flex-column gap-1 w-6">
          <ValidationError
            :is-required="true"
            :message="v$.record.currency.$errors[0]?.$message"
          >
            <label>Currency</label>
          </ValidationError>
          <Select
            v-model="record.currency"
            :options="availableCurrencies"
            size="small"
            placeholder="Select currency"
            :readonly="isReadOnly"
            :disabled="isReadOnly"
          />
        </div>
      </div>

      <div class="flex flex-row w-full gap-3">
        <div class="flex flex-column gap-1 w-6">
          <ValidationError
            :is-required="true"
            :message="v$.record.price_per_unit.$errors[0]?.$message"
          >
            <label>Price per unit</label>
          </ValidationError>
          <InputNumber
            v-model="pricePerUnitNumber"
            size="small"
            mode="currency"
            :currency="record.currency"
            locale="de-DE"
            :placeholder="getCurrencyPlaceholder(record.currency)"
            :readonly="isReadOnly"
            :disabled="isReadOnly"
            fluid
          />
        </div>

        <div class="flex flex-column gap-1 w-6">
          <ValidationError
            :is-required="false"
            :message="v$.record.fee.$errors[0]?.$message"
          >
            <label>Fee</label>
          </ValidationError>
          <InputNumber
            v-model="feeNumber"
            size="small"
            mode="currency"
            :currency="record.currency"
            locale="de-DE"
            :min-fraction-digits="2"
            :max-fraction-digits="record.holding?.investment_type === 'crypto' ? 6 : 2"
            :placeholder="getCurrencyPlaceholder(record.currency)"
            :readonly="isReadOnly"
            :disabled="isReadOnly"
            fluid
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
            size="small"
            placeholder="Describe transaction"
          />
        </div>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <div class="flex flex-column w-full gap-2">
          <Button
            class="main-button"
            :label="(mode == 'create' ? 'Insert' : 'Update') + ' transaction'"
            style="height: 42px"
            @click="manageRecord"
          />
          <Button
            v-if="mode == 'update'"
            label="Delete transaction"
            class="delete-button"
            style="height: 42px"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
