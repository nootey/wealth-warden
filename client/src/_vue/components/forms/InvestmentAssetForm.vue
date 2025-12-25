<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import type { Account } from "../../../models/account_models.ts";
import type {
  InvestmentAsset,
  TickerData,
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
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const apiPrefix = "investments";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const accountStore = useAccountStore();

const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const isReadOnly = computed(() => props.mode === "update");

const accounts = ref<Account[]>([]);
const record = ref<InvestmentAsset>(initData());
const filteredAccounts = ref<Account[]>([]);
const infoTooltipRef = ref<any>(null);

const quantitytRef = computed({
  get: () => record.value.quantity,
  set: (v) => (record.value.quantity = v),
});
const { number: quantityNumber } = currencyHelper.useMoneyField(
  quantitytRef,
  2,
);

const investmentTypes = ref<string[]>(["Crypto", "Stock", "ETF"]);

const selectedInvestmentType = ref<string>(
  investmentTypes.value.find((i) => i === "Crypto") ?? "ETF",
);

const availableAccounts = computed(() => {
  const typeMap: Record<string, string[]> = {
    crypto: ["wallet", "exchange"],
    stock: ["brokerage", "retirement", "pension"],
    etf: ["brokerage", "retirement", "pension", "mutual_fund"],
  };

  const allowedSubtypes =
    typeMap[selectedInvestmentType.value.toLowerCase()] || [];

  return accounts.value.filter((acc) =>
    allowedSubtypes.includes(acc.account_type.sub_type),
  );
});

const tickerData = ref<TickerData>({
  name: "",
  exchange: "",
  currency: "",
});

const rules = {
  record: {
    name: {
      required,
      $autoDirty: true,
    },
    account: {
      required,
      $autoDirty: true,
    },
    investment_type: {
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
  },
  tickerData: {
    name: {
      required,
      $autoDirty: true,
    },
    exchange: {
      $autoDirty: true,
    },
    currency: {
      $autoDirty: true,
    },
  },
};

const v$ = useVuelidate(rules, { record, tickerData });

onMounted(async () => {
  accounts.value = await accountStore.getAllAccounts(true, true);

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

function initData(): InvestmentAsset {
  return {
    account: null,
    investment_type: "crypto",
    name: "",
    ticker: "",
    quantity: "0",
  };
}

const searchAccount = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredAccounts.value = [...availableAccounts.value];
    } else {
      filteredAccounts.value = availableAccounts.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
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

    // Parse ticker based on format
    if (data.ticker) {
      if (data.ticker.includes("-")) {
        // Crypto: "BTC-USD"
        const [name, currency] = data.ticker.split("-");
        tickerData.value = { name, exchange: "", currency };
      } else if (data.ticker.includes(".")) {
        // Stock/ETF: "IWDA.L"
        const [name, exchange] = data.ticker.split(".");
        tickerData.value = { name, exchange, currency: "" };
      } else {
        tickerData.value = { name: data.ticker, exchange: "", currency: "" };
      }
    }

    record.value = {
      ...initData(),
      ...data,
    };

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function manageRecord() {
  if (!(await isRecordValid())) return;
  if (!record.value.account) return;

  loading.value = true;
  let ticker = tickerData.value.name;

  if (selectedInvestmentType.value.toLowerCase() === "crypto") {
    // Crypto: "BTC-USD"
    const currency = tickerData.value.currency
      ? tickerData.value.currency.toUpperCase()
      : "USD";
    ticker = `${ticker}-${currency}`;
  } else {
    // Stock/ETF: "IWDA.L"
    if (tickerData.value.exchange) {
      ticker = `${ticker}.${tickerData.value.exchange.toUpperCase()}`;
    }
  }

  const recordData = {
    account_id: record.value.account.id,
    investment_type: selectedInvestmentType.value.toLowerCase(),
    quantity: record.value.quantity,
    name: record.value.name,
    ticker: ticker,
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

function toggleInfoPopup(event: any) {
  infoTooltipRef.value.toggle(event);
}

async function deleteConfirmation(id: number) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete asset: "${id}". This action is not reversible!`,
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
  <Popover
    ref="infoTooltipRef"
    class="rounded-popover"
    :style="{ width: '300px' }"
    :breakpoints="{ '301px': '90vw' }"
  >
    <div class="flex flex-column gap-3 p-2">
      <div class="flex flex-column gap-2">
        <div class="text-sm font-semibold">Crypto format:</div>
        <div class="flex flex-column gap-1 text-sm">
          <div class="flex justify-content-between">
            <span class="font-medium">BTC-USD</span>
            <span class="text-color-secondary">Bitcoin in USD</span>
          </div>
        </div>
        <div class="text-xs text-color-secondary">
          Default currency is USD if not specified.
        </div>
      </div>

      <div class="flex flex-column gap-2">
        <div class="text-sm font-semibold">Stocks/ETF format:</div>
        <div class="text-sm">Supported Exchanges:</div>
        <div class="flex flex-column gap-1 text-sm">
          <div class="flex justify-content-between">
            <span class="font-medium">L</span>
            <span class="text-color-secondary">London (LSE)</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">AS</span>
            <span class="text-color-secondary">Amsterdam (Euronext)</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">PA</span>
            <span class="text-color-secondary">Paris (Euronext)</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">DE</span>
            <span class="text-color-secondary">Germany (XETRA)</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">F</span>
            <span class="text-color-secondary">Frankfurt</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">TO</span>
            <span class="text-color-secondary">Toronto (TSX)</span>
          </div>
          <div class="flex justify-content-between">
            <span class="font-medium">AX</span>
            <span class="text-color-secondary">Australia (ASX)</span>
          </div>
        </div>
        <div class="text-xs text-color-secondary">
          Leave empty for US stocks (NYSE/NASDAQ)
        </div>
      </div>
    </div>
  </Popover>

  <div v-if="!loading" class="flex flex-column gap-3 p-1">
    <div v-if="!isReadOnly" class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column">
        <SelectButton
          v-model="selectedInvestmentType"
          style="font-size: 0.875rem"
          size="small"
          :options="investmentTypes"
          :allow-empty="false"
        />
      </div>
    </div>

    <span
      v-if="isReadOnly"
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      This is a read only view. Due to the complexity of re-calculating the
      financial impact of this record, most fields can not be updated.
    </span>

    <span
      v-if="isReadOnly"
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      If you wish to make changes, delete the asset and create a new one. That
      will also delete all related trades, and reverse their effects.
    </span>

    <div v-if="mode === 'update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>Investment type</label>
        <span style="color: var(--text-secondary)">{{
          record.investment_type
        }}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>Average</label>
        <span style="color: var(--text-secondary)">{{
          vueHelper.displayAsCurrency(record.average_buy_price!)
        }}</span>
      </div>
    </div>

    <div v-if="mode === 'update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>Value at buy</label>
        <span style="color: var(--text-secondary)">{{
          vueHelper.displayAsCurrency(record.value_at_buy!)
        }}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>Current value</label>
        <span style="color: var(--text-secondary)">{{
          vueHelper.displayAsCurrency(record.current_value!)
        }}</span>
      </div>
    </div>

    <div v-if="mode === 'update'" class="flex flex-row w-full gap-3">
      <div class="flex flex-column gap-1 w-6">
        <label>Current price</label>
        <span style="color: var(--text-secondary)">{{
          vueHelper.displayAsCurrency(record.current_price!)
        }}</span>
      </div>
      <div class="flex flex-column gap-1 w-6">
        <label>Last price update</label>
        <span style="color: var(--text-secondary)">{{
          dateHelper.formatDate(record.last_price_update!, true)
        }}</span>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="v$.record.account.$errors[0]?.$message"
        >
          <label>Account</label>
        </ValidationError>
        <AutoComplete
          v-model="record.account"
          size="small"
          :suggestions="filteredAccounts"
          option-label="name"
          option-value="id"
          data-key="id"
          force-selection
          placeholder="Select account"
          dropdown
          :readonly="isReadOnly"
          :disabled="isReadOnly"
          @complete="searchAccount"
        />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="v$.record.name.$errors[0]?.$message"
        >
          <label>Name</label>
        </ValidationError>
        <InputText
          v-model="record.name"
          size="small"
          placeholder="Input asset name"
        />
      </div>
    </div>

    <div
      v-if="!isReadOnly"
      class="flex flex-row w-full align-items-center gap-2"
    >
      <i
        class="pi pi-info-circle hover-icon text-sm"
        @click="toggleInfoPopup"
      ></i>
      <span style="color: var(--text-secondary)">Formatting guide</span>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="v$.tickerData.name.$errors[0]?.$message"
        >
          <label>Ticker</label>
        </ValidationError>
        <InputText
          v-model="tickerData.name"
          size="small"
          placeholder="Input ticker"
          :readonly="isReadOnly"
          :disabled="isReadOnly"
        />
      </div>
    </div>

    <div class="flex flex-row w-full gap-2">
      <div
        v-if="selectedInvestmentType.toLowerCase() !== 'crypto'"
        class="flex flex-column gap-1 w-full"
      >
        <ValidationError
          :is-required="false"
          :message="v$.tickerData.exchange.$errors[0]?.$message"
        >
          <label>Exchange</label>
        </ValidationError>
        <InputText
          v-model="tickerData.exchange"
          size="small"
          placeholder="Input exchange"
          :readonly="isReadOnly"
          :disabled="isReadOnly"
        />
      </div>
      <div v-else class="flex flex-column gap-1 w-full">
        <ValidationError
          :is-required="false"
          :message="v$.tickerData.currency.$errors[0]?.$message"
        >
          <label>Currency</label>
        </ValidationError>
        <InputText
          v-model="tickerData.currency"
          size="small"
          placeholder="Input currency"
          :readonly="isReadOnly"
          :disabled="isReadOnly"
        />
      </div>
    </div>

    <div v-if="mode === 'update'" class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
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
        />
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />

  <div class="flex flex-column mt-3 gap-3">
    <Button
      class="main-button"
      :label="(mode == 'create' ? 'Insert' : 'Update') + ' asset'"
      style="height: 42px"
      :disabled="loading"
      @click="manageRecord"
    />
    <Button
      v-if="mode == 'update'"
      label="Delete asset"
      class="delete-button"
      style="height: 42px"
      :disabled="loading"
      @click="deleteConfirmation(record.id!)"
    />
  </div>
</template>

<style scoped></style>
