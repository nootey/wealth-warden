<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import type {
  InvestmentAsset,
  InvestmentTrade,
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
import { useInvestmentStore } from "../../../services/stores/investment_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import AuditTrail from "../base/AuditTrail.vue";
import type { UserSettings } from "../../../models/settings_models.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const apiPrefix = "investments/trades";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const investmentStore = useInvestmentStore();
const settingsStore = useSettingsStore();

const loading = ref(false);
const isReadOnly = computed(() => props.mode === "update");

const confirm = useConfirm();
const { hasPermission } = usePermissions();

const assets = ref<InvestmentAsset[]>([]);
const record = ref<InvestmentTrade>(initData());
const filteredAssets = ref<InvestmentAsset[]>([]);
const userSettings = ref<UserSettings>();

const tradeTypes = ref<string[]>(["Buy", "Sell"]);

const selectedTradeType = ref<string>(
  tradeTypes.value.find((i) => i === "Buy") ?? "Sell",
);

const availableCurrencies = ref<string[]>(["USD", "EUR"]);

const quantityRef = computed({
  get: () => record.value.quantity,
  set: (v) => (record.value.quantity = v),
});
const { number: quantityNumber } = currencyHelper.useMoneyField(quantityRef, 6);

const feeRef = computed({
  get: () => record.value.fee,
  set: (v) => (record.value.fee = v),
});
const { number: feeNumber } = currencyHelper.useMoneyField(feeRef, 6);

const pricePerUnitRef = computed({
  get: () => record.value.price_per_unit,
  set: (v) => (record.value.price_per_unit = v),
});
const { number: pricePerUnitNumber } = currencyHelper.useMoneyField(
  pricePerUnitRef,
  6,
);

const rules = {
  record: {
    asset: {
      required,
      $autoDirty: true,
    },
    txn_date: {
      required,
      $autoDirty: true,
    },
    trade_type: {
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
  },
};

const v$ = useVuelidate(rules, { record });

onMounted(async () => {
  await getSettings();
  assets.value = await investmentStore.getAllAssets();

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

function initData(): InvestmentTrade {
  return {
    asset: null,
    txn_date: dayjs().toDate(),
    trade_type: "buy",
    quantity: "",
    fee: "0",
    price_per_unit: "",
    currency: "USD",
    description: "",
  };
}

async function getSettings() {
  try {
    const res = await settingsStore.getUserSettings();
    userSettings.value = res.data;
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

function getCurrencyPlaceholder(currency: string) {
  if (record.value.asset?.investment_type === "crypto") return "0";

  const symbols: Record<string, string> = {
    USD: "$",
    EUR: "€",
    GBP: "£",
  };
  return `0,00 ${symbols[currency] || currency}`;
}

const searchAsset = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredAssets.value = [...assets.value];
    } else {
      filteredAssets.value = assets.value.filter((record) => {
        return record.ticker
          .toLowerCase()
          .startsWith(event.query.toLowerCase());
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
    const data = await sharedStore.getRecordByID(apiPrefix, id);

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
  if (!record.value.asset) return;

  loading.value = true;

  const txn_date = dateHelper.mergeDateWithCurrentTime(
    dayjs(record.value.txn_date).format("YYYY-MM-DD"),
    userSettings.value?.timezone || "UTC",
  );

  const recordData = {
    asset_id: record.value.asset.id,
    trade_type: selectedTradeType.value.toLowerCase(),
    txn_date: txn_date,
    quantity: record.value.quantity,
    price_per_unit: record.value.price_per_unit,
    fee: record.value.fee,
    currency: record.value.currency.toUpperCase(),
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
  } finally {
    loading.value = false;
  }
}

async function deleteConfirmation(id: number) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete trade: "${id}". This action is not reversible!`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id),
  });
}

async function deleteRecord(id: number) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  loading.value = true;

  try {
    let response = await sharedStore.deleteRecord(apiPrefix, id);
    toastStore.successResponseToast(response);
    emit("completeDelete");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div v-if="!loading" class="flex flex-column gap-3 p-1">
    <div v-if="!isReadOnly" class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column">
        <SelectButton
          v-model="selectedTradeType"
          style="font-size: 0.875rem"
          size="small"
          :options="tradeTypes"
          :allow-empty="false"
          :readonly="isReadOnly"
          :disabled="isReadOnly"
        />
      </div>
    </div>

    <div v-if="isReadOnly" class="flex flex-column gap-2">
      <h4>Info</h4>
      <span class="text-sm" style="color: var(--text-secondary)">
        This is a read only view. Due to the complexity of re-calculating the
        financial impact of the trade, most fields can not be updated.
      </span>
      <span class="text-sm" style="color: var(--text-secondary)">
        If you wish to make changes, delete the trade and create a new one.
      </span>
    </div>

    <div v-if="isReadOnly" class="flex flex-column gap-2">
      <h4>Financial details</h4>
      <div class="flex flex-row w-full gap-3">
        <div class="flex flex-column gap-1 w-6">
          <label>Trade type</label>
          <span style="color: var(--text-secondary)">{{
            record.trade_type
          }}</span>
        </div>
        <div class="flex flex-column gap-1 w-6">
          <label>USD exchange rate</label>
          <span style="color: var(--text-secondary)">{{
            record.exchange_rate_to_usd
          }}</span>
        </div>
      </div>

      <div class="flex flex-row w-full gap-3">
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">Value at buy</label>
          <span class="text-sm" style="color: var(--text-secondary)">{{
            vueHelper.displayAsCurrency(record.value_at_buy!, record.currency)
          }}</span>
        </div>
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">{{
            record.trade_type === "buy" ? "Current value" : "Value at sell"
          }}</label>
          <span class="text-sm" style="color: var(--text-secondary)">
            {{
              vueHelper.displayAsCurrency(
                record.trade_type === "buy"
                  ? record.current_value!
                  : record.realized_value!,
                record.currency,
              )
            }}</span
          >
        </div>
      </div>

      <div
        v-if="record.trade_type === 'sell'"
        class="flex flex-row w-full gap-3"
      >
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">What if</label>
          <span class="text-sm" style="color: var(--text-secondary)"
            >You haven't sold</span
          >
        </div>
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">Current market value</label>
          <span class="text-sm" style="color: var(--text-secondary)">{{
            vueHelper.displayAsCurrency(record.current_value!, record.currency)
          }}</span>
        </div>
      </div>

      <div class="flex flex-row w-full gap-3">
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">P&L Raw</label>
          <span class="text-sm" style="color: var(--text-secondary)">{{
            vueHelper.displayAsCurrency(record.profit_loss!, record.currency)
          }}</span>
        </div>
        <div class="flex flex-column gap-1 w-6">
          <label class="text-sm">P&L Percentage</label>
          <span class="text-sm" style="color: var(--text-secondary)">{{
            vueHelper.displayAsPercentage(record.profit_loss_percent!)
          }}</span>
        </div>
      </div>
    </div>

    <h4 v-if="isReadOnly">Trade details</h4>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="v$.record.asset.$errors[0]?.$message"
        >
          <label>Asset</label>
        </ValidationError>
        <AutoComplete
          v-model="record.asset"
          size="small"
          :suggestions="filteredAssets"
          option-label="name"
          data-key="id"
          force-selection
          placeholder="Select asset"
          dropdown
          :readonly="isReadOnly"
          :disabled="isReadOnly"
          @complete="searchAsset"
        >
          <template #option="slotProps">
            <div class="flex align-items-center gap-2">
              <span class="font-semibold">{{ slotProps.option.name }}</span>
              <span class="text-color-secondary">{{
                slotProps.option.ticker
              }}</span>
            </div>
          </template>

          <template #chip="slotProps">
            <div class="flex align-items-center gap-2">
              <span class="font-semibold">{{ slotProps.value.name }}</span>
              <span class="text-color-secondary">{{
                slotProps.value.ticker
              }}</span>
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
          :min-fraction-digits="2"
          :max-fraction-digits="
            record.asset?.investment_type === 'crypto' ? 6 : 2
          "
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
          :mode="
            record.asset?.investment_type === 'crypto' ? 'decimal' : 'currency'
          "
          :currency="record.currency"
          locale="de-DE"
          :min-fraction-digits="2"
          :max-fraction-digits="
            record.asset?.investment_type === 'crypto' ? 6 : 2
          "
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
          placeholder="Describe trade"
        />
      </div>
    </div>

    <h4 v-if="isReadOnly">Auditing</h4>
    <div v-if="isReadOnly" class="flex flex-row gap-2 w-full">
      <AuditTrail
        :record-id="props.recordId!"
        :events="['create', 'update']"
        category="investment_trade"
      />
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />

  <div class="flex flex-column w-full gap-3 mt-3">
    <Button
      class="main-button"
      :label="(mode == 'create' ? 'Insert' : 'Update') + ' trade'"
      style="height: 42px"
      :disabled="loading"
      @click="manageRecord"
    />
    <Button
      v-if="mode == 'update'"
      label="Delete trade"
      class="delete-button"
      style="height: 42px"
      :disabled="loading"
      @click="deleteConfirmation(record.id!)"
    />
  </div>
</template>

<style scoped></style>
