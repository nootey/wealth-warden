<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import type {
  InvestmentAsset,
  InvestmentTrade,
} from "../../../models/investment_models.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import { required } from "@regle/rules";
import {
  decimalMax,
  decimalMin,
  decimalValid,
} from "../../../validators/currency.ts";
import { useRegle } from "@regle/core";
import ValidationError from "../validation/ValidationError.vue";
import dayjs from "dayjs";
import dateHelper from "../../../utils/date_helper.ts";
import { useInvestmentStore } from "../../../services/stores/investment_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import BaseForm from "../base/BaseForm.vue";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import AuditTrail from "../base/AuditTrail.vue";
import type { UserSettings } from "../../../models/settings_models.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import searchHelper from "../../../utils/search_helper.ts";
import Decimal from "decimal.js";

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

const isCrypto = computed(
  () => record.value.asset?.investment_type === "crypto",
);

const pnlClass = computed(() => {
  const pnl = new Decimal(record.value.profit_loss || 0);
  if (pnl.isZero()) return "text-muted";
  return pnl.isPositive() ? "text-gain" : "text-loss";
});

const factTiles = computed(() => {
  const r = record.value;
  return [
    { label: "Quantity", value: new Decimal(r.quantity || 0).toString() },
    {
      label: "Price per unit",
      value: vueHelper.displayAssetPrice(
        r.price_per_unit,
        r.asset?.investment_type,
        r.currency,
      ),
    },
    {
      label: "Fee",
      value: isCrypto.value
        ? new Decimal(r.fee || 0).toString()
        : vueHelper.displayAsCurrency(r.fee, r.currency),
    },
    {
      label: "Value at buy",
      value: vueHelper.displayAsCurrency(r.value_at_buy!, r.currency),
    },
    {
      label: "USD rate",
      value: r.exchange_rate_to_usd
        ? new Decimal(r.exchange_rate_to_usd).toString()
        : null,
    },
    { label: "Currency", value: r.currency },
  ];
});

const taxTiles = computed(() => {
  const r = record.value;
  const tax = r.tax_info;
  if (!tax) return [];

  let taxFree: string | null = null;
  if (tax.days_until_tax_free === 0) taxFree = "Now";
  else if (tax.days_until_tax_free != null)
    taxFree = `${tax.days_until_tax_free} days`;

  return [
    {
      label: "After-tax P&L",
      value: vueHelper.displayAsCurrency(tax.taxable_profit, r.currency),
    },
    {
      label: "Tax bracket",
      value:
        tax.taxable_percent != null ? `${Number(tax.taxable_percent)} %` : null,
    },
    { label: "Days held", value: String(tax.days_held) },
    { label: "Tax-free in", value: taxFree },
  ];
});

const rules = {
  asset: {
    $self: { required },
  },
  txn_date: {
    required,
  },
  trade_type: {
    required,
  },
  quantity: {
    required,
    decimalValid,
    decimalMin: decimalMin(0),
    decimalMax: decimalMax(1_000_000_000),
  },
  fee: {
    required,
    decimalValid,
    decimalMin: decimalMin(0),
    decimalMax: decimalMax(1_000_000_000),
  },
  price_per_unit: {
    required,
    decimalValid,
    decimalMin: decimalMin(0),
    decimalMax: decimalMax(1_000_000_000),
  },
  currency: {
    required,
  },
  description: {},
};

const { r$ } = useRegle(record, rules);

onMounted(async () => {
  await getSettings();
  assets.value = await investmentStore.getAllAssets();

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

watch(
  () => record.value.asset,
  (newAsset) => {
    if (newAsset && newAsset.currency) {
      record.value.currency = newAsset.currency;
    }
  },
);

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
  filteredAssets.value = searchHelper.filterByQuery(
    assets.value,
    event.query,
    (record) => [record.name, record.ticker],
  );
};

async function isRecordValid() {
  const { valid: isValid } = await r$.$validate();
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

    r$.$reset();
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
  <BaseForm
    v-if="!loading"
    class="flex flex-col gap-4 p-1"
    :disabled="isReadOnly || loading"
    @submit="manageRecord"
  >
    <template v-if="isReadOnly">
      <div class="flex flex-row items-center gap-3">
        <div
          class="flex items-center justify-center w-10 h-10 shrink-0 rounded-xl bg-sunken text-muted"
        >
          <i
            :class="[
              'pi',
              record.trade_type === 'buy'
                ? 'pi-arrow-down-left'
                : 'pi-arrow-up-right',
            ]"
          />
        </div>
        <div class="flex flex-col gap-1 min-w-0">
          <span class="text-lg font-medium tracking-tight text-ink truncate">
            {{ record.asset?.name }}
          </span>
          <div class="flex flex-row flex-wrap items-center gap-2 text-xs">
            <span
              class="rounded-full px-2 py-0.5 font-medium bg-sunken"
              :class="record.trade_type === 'buy' ? 'text-gain' : 'text-loss'"
            >
              {{ vueHelper.capitalize(record.trade_type) }}
            </span>
            <span
              v-if="record.asset?.ticker"
              class="rounded-full px-2 py-0.5 font-medium bg-sunken text-ink"
            >
              {{ record.asset.ticker }}
            </span>
            <span class="text-faint">
              {{ dateHelper.formatDate(record.txn_date, false) }}
            </span>
          </div>
        </div>
      </div>

      <span class="text-xs text-muted">
        Mostly read only due to re-calculations. To change it, delete the trade
        and create a new one.
      </span>

      <div
        class="grid gap-2"
        :class="record.trade_type === 'sell' ? 'grid-cols-3' : 'grid-cols-2'"
      >
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
          <span class="label">
            {{
              record.trade_type === "buy" ? "Current value" : "Value at sell"
            }}
          </span>
          <span class="text-xl font-medium tracking-tight text-ink truncate">
            {{
              vueHelper.displayAsCurrency(
                record.trade_type === "buy"
                  ? record.current_value!
                  : record.realized_value!,
                record.currency,
              )
            }}
          </span>
        </div>
        <div
          v-if="record.trade_type === 'sell'"
          class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0"
        >
          <span class="label">If held</span>
          <span class="text-xl font-medium tracking-tight text-ink truncate">
            {{
              vueHelper.displayAsCurrency(
                record.current_value!,
                record.currency,
              )
            }}
          </span>
        </div>
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
          <span class="label">P&L</span>
          <span
            class="text-xl font-medium tracking-tight truncate"
            :class="pnlClass"
          >
            {{
              vueHelper.displayAsCurrency(record.profit_loss!, record.currency)
            }}
            <span class="text-sm">
              ·
              {{ vueHelper.displayAsPercentage(record.profit_loss_percent!) }}
            </span>
          </span>
        </div>
      </div>

      <div id="trade-tiles" class="grid grid-cols-3 gap-2">
        <div
          v-for="tile in factTiles"
          :key="tile.label"
          class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0"
        >
          <span class="text-xs text-muted">{{ tile.label }}</span>
          <span class="text-base font-medium tracking-tight text-ink truncate">
            {{ tile.value ?? "-" }}
          </span>
        </div>
      </div>

      <section
        v-if="record.tax_info"
        class="flex flex-col gap-3 rounded-2xl border border-line p-4"
      >
        <span class="label">Tax</span>
        <div class="grid grid-cols-2 gap-2">
          <div
            v-for="tile in taxTiles"
            :key="tile.label"
            class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0"
          >
            <span class="text-xs text-muted">{{ tile.label }}</span>
            <span
              class="text-base font-medium tracking-tight text-ink truncate"
            >
              {{ tile.value ?? "-" }}
            </span>
          </div>
        </div>
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Details</span>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="false"
            :message="r$.description.$errors[0]"
          >
            <label>Description</label>
          </ValidationError>
          <InputText
            v-model="record.description"
            size="small"
            placeholder="Describe trade"
          />
        </div>
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Auditing</span>
        <AuditTrail
          :record-id="props.recordId!"
          :events="['create', 'update']"
          :categories="['investment_trade']"
        />
      </section>
    </template>

    <template v-else>
      <div class="flex flex-row w-full justify-center">
        <SelectButton
          v-model="selectedTradeType"
          size="small"
          class="text-sm"
          :options="tradeTypes"
          :allow-empty="false"
        />
      </div>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Trade</span>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="r$.asset.$errors.$self?.[0]"
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
            @complete="searchAsset"
          >
            <template #option="slotProps">
              <div class="flex items-center gap-2">
                <span class="font-semibold">{{ slotProps.option.name }}</span>
                <span class="text-muted">{{ slotProps.option.ticker }}</span>
              </div>
            </template>

            <template #chip="slotProps">
              <div class="flex items-center gap-2">
                <span class="font-semibold">{{ slotProps.value.name }}</span>
                <span class="text-muted">{{ slotProps.value.ticker }}</span>
              </div>
            </template>
          </AutoComplete>
        </div>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="r$.txn_date.$errors[0]"
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
          />
        </div>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="false"
            :message="r$.description.$errors[0]"
          >
            <label>Description</label>
          </ValidationError>
          <InputText
            v-model="record.description"
            size="small"
            placeholder="Describe trade"
          />
        </div>
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Amounts</span>
        <div class="flex flex-row w-full gap-4">
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="r$.quantity.$errors[0]"
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
              fluid
            />
          </div>
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="r$.currency.$errors[0]"
            >
              <label>Currency</label>
            </ValidationError>
            <Select
              v-model="record.currency"
              :options="availableCurrencies"
              size="small"
              placeholder="Select currency"
            />
          </div>
        </div>
        <div class="flex flex-row w-full gap-4">
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="r$.price_per_unit.$errors[0]"
            >
              <label>Price per unit</label>
            </ValidationError>
            <InputNumber
              v-model="pricePerUnitNumber"
              size="small"
              mode="currency"
              :currency="record.currency"
              :locale="vueHelper.getCurrencyLocale(record.currency)"
              :min-fraction-digits="2"
              :max-fraction-digits="
                record.asset?.investment_type === 'crypto' ? 6 : 2
              "
              :placeholder="getCurrencyPlaceholder(record.currency)"
              fluid
            />
          </div>
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError :is-required="false" :message="r$.fee.$errors[0]">
              <label>Fee</label>
            </ValidationError>
            <InputNumber
              v-model="feeNumber"
              size="small"
              :mode="
                record.asset?.investment_type === 'crypto'
                  ? 'decimal'
                  : 'currency'
              "
              :currency="record.currency"
              locale="de-DE"
              :min-fraction-digits="2"
              :max-fraction-digits="
                record.asset?.investment_type === 'crypto' ? 6 : 2
              "
              :placeholder="getCurrencyPlaceholder(record.currency)"
              fluid
            />
          </div>
        </div>
      </section>
    </template>
  </BaseForm>
  <ShowLoading v-else :num-fields="5" />

  <div class="flex flex-col w-full gap-4 mt-4">
    <Button
      class="main-button"
      :label="(mode == 'create' ? 'Insert' : 'Update') + ' trade'"
      :disabled="loading"
      @click="manageRecord"
    />
    <Button
      v-if="mode == 'update'"
      label="Delete trade"
      class="delete-button"
      :disabled="loading"
      @click="deleteConfirmation(record.id!)"
    />
  </div>
</template>

<style scoped>
@media (max-width: 640px) {
  #trade-tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
