<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useWsStore } from "../../../services/stores/ws_store.ts";
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from "vue";
import type { Account } from "../../../models/account_models.ts";
import type { AssetPnLPayload } from "../../../models/ws_models.ts";
import type {
  InvestmentAsset,
  InvestmentType,
  TickerData,
} from "../../../models/investment_models.ts";
import { required } from "@regle/rules";
import {
  decimalMax,
  decimalMin,
  decimalValid,
} from "../../../validators/currency.ts";
import { useRegle } from "@regle/core";
import ValidationError from "../validation/ValidationError.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import BaseForm from "../base/BaseForm.vue";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import AuditTrail from "../base/AuditTrail.vue";
import InvestmentAssetWidget from "../../features/widgets/InvestmentAssetWidget.vue";
import InvestmentIncomeForm from "./InvestmentIncomeForm.vue";
import InvestmentIncomePaginated from "../data/InvestmentIncomePaginated.vue";
import searchHelper from "../../../utils/search_helper.ts";
import tickerHelper from "../../../utils/ticker_helper.ts";
import Decimal from "decimal.js";

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
const wsStore = useWsStore();

const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const isReadOnly = computed(() => props.mode === "update");
const showIncomeForm = ref(false);
const incomeListRef = ref<InstanceType<
  typeof InvestmentIncomePaginated
> | null>(null);

const accounts = ref<Account[]>([]);
const record = ref<InvestmentAsset>(initData());
const filteredAccounts = ref<Account[]>([]);
const infoTooltipRef = ref<any>(null);

const investmentTypes = ref<string[]>(["Crypto", "Stock", "ETF"]);

const selectedInvestmentType = ref<string>(
  investmentTypes.value.find((i) => i === "Crypto") ?? "ETF",
);

const availableCurrencies = ref<string[]>(["USD", "EUR"]);

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

const hasHoldings = computed(() => parseFloat(record.value.quantity) > 0);

const investmentTypeLabel = computed(() =>
  record.value.investment_type === "etf"
    ? "ETF"
    : vueHelper.capitalize(record.value.investment_type),
);

const pnlClass = computed(() => {
  const pnl = new Decimal(record.value.profit_loss || 0);
  if (pnl.isZero()) return "text-muted";
  return pnl.isPositive() ? "text-gain" : "text-loss";
});

const factTiles = computed(() => {
  const r = record.value;
  const tiles = [
    {
      label: "Quantity",
      value: new Decimal(r.quantity || 0).toString(),
    },
    {
      label: "Average price",
      value: vueHelper.displayAsCurrency(r.average_buy_price!, r.currency),
    },
    {
      label: "Current price",
      value: vueHelper.displayAssetPrice(
        r.latest_price!,
        r.investment_type,
        r.latest_price_currency!,
      ),
    },
    {
      label: "Value at buy",
      value: vueHelper.displayAsCurrency(r.value_at_buy!, r.currency),
    },
    {
      label: "Fees",
      value: vueHelper.displayAsCurrency(r.total_fees!, r.currency),
    },
    {
      label: "Price updated",
      value: dateHelper.formatDate(r.last_price_update!, true),
    },
  ];

  if (r.tax_summary) {
    tiles.push(
      {
        label: "After-tax P&L",
        value: vueHelper.displayAsCurrency(
          r.tax_summary.after_tax_pnl,
          r.currency,
        ),
      },
      {
        label: "Est. tax due",
        value: vueHelper.displayAsCurrency(
          r.tax_summary.estimated_tax_due,
          r.currency,
        ),
      },
      {
        label: "Tax bracket",
        value:
          r.tax_summary.taxable_percent !== null
            ? `${Number(r.tax_summary.taxable_percent)} %`
            : null,
      },
    );
  }

  return tiles;
});

const detailTiles = computed(() => [
  { label: "Account", value: record.value.account?.name },
  { label: "Ticker", value: tickerData.value.name },
  { label: "Currency", value: record.value.currency },
  { label: "Exchange", value: tickerData.value.exchange },
]);

const tickerData = ref<TickerData>({
  name: "",
  exchange: "",
  currency: "",
});

const rules = {
  record: {
    name: {
      required,
    },
    account: {
      $self: { required },
    },
    investment_type: {
      required,
    },
    quantity: {
      required,
      decimalValid,
      decimalMin: decimalMin(0),
      decimalMax: decimalMax(1_000_000_000),
    },
    currency: {
      required,
    },
  },
  tickerData: {
    name: {
      required,
    },
    exchange: {},
  },
};

// `reactive` unwraps the refs, so writes from the v-models stay bound
const validationState = reactive({ record, tickerData });

const { r$ } = useRegle(validationState, rules);

const unsubscribers: (() => void)[] = [];

onMounted(async () => {
  accounts.value = await accountStore.getAllAccounts(true, true);

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }

  unsubscribers.push(
    wsStore.on("asset.pnl_synced", (payload) => {
      const { asset_id, account_id } = (payload ?? {}) as AssetPnLPayload;
      const id = record.value.id;
      if (!id) return;
      if (
        asset_id === id ||
        (account_id != null && account_id === record.value.account?.id)
      ) {
        loadRecord(id);
      }
    }),
  );
});

onUnmounted(() => unsubscribers.forEach((unsubscribe) => unsubscribe()));

function initData(): InvestmentAsset {
  return {
    account: null,
    investment_type: "crypto",
    name: "",
    ticker: "",
    quantity: "0",
    currency: "USD",
  };
}

const searchAccount = (event: { query: string }) => {
  filteredAccounts.value = searchHelper.filterByQuery(
    availableAccounts.value,
    event.query,
    (record) => [record.name],
    { sort: true },
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

    if (data.ticker) {
      tickerData.value = tickerHelper.parseTicker(
        data.ticker,
        data.investment_type,
      );
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

  const investmentType =
    selectedInvestmentType.value.toLowerCase() as InvestmentType;
  const ticker = tickerHelper.buildTicker(
    {
      name: tickerData.value.name,
      exchange: tickerData.value.exchange ?? "",
      currency: record.value.currency,
    },
    investmentType,
  );

  const recordData = {
    account_id: record.value.account.id,
    investment_type: investmentType,
    quantity: record.value.quantity,
    name: record.value.name,
    ticker: ticker,
    currency: record.value.currency.toUpperCase(),
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

    r$.record.$reset();
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

async function syncAssetPrice(id: number | null) {
  if (!id) return;

  try {
    let response = await accountStore.syncAssetPnL(id);
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function onIncomeAdded(): void {
  showIncomeForm.value = false;
  incomeListRef.value?.refresh();
}

async function syncAssetAccountBalance(acc_id: number | null) {
  if (!acc_id) return;

  try {
    let response = await accountStore.syncAssetAccountBalance(acc_id);
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
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
    <div class="flex flex-col gap-4 p-2">
      <div class="flex flex-col gap-2">
        <div class="text-sm font-semibold">Crypto format:</div>
        <div class="flex flex-col gap-1 text-sm">
          <div class="flex justify-between">
            <span class="font-medium">BTC-USD</span>
            <span class="text-muted-color">Bitcoin in USD</span>
          </div>
        </div>
        <div class="text-xs text-muted-color">
          Default currency is USD if not specified.
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <div class="text-sm font-semibold">Stocks/ETF format:</div>
        <div class="text-sm">Supported Exchanges:</div>
        <div class="flex flex-col gap-1 text-sm">
          <div class="flex justify-between">
            <span class="font-medium">L</span>
            <span class="text-muted-color">London (LSE)</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">AS</span>
            <span class="text-muted-color">Amsterdam (Euronext)</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">PA</span>
            <span class="text-muted-color">Paris (Euronext)</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">DE</span>
            <span class="text-muted-color">Germany (XETRA)</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">F</span>
            <span class="text-muted-color">Frankfurt</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">TO</span>
            <span class="text-muted-color">Toronto (TSX)</span>
          </div>
          <div class="flex justify-between">
            <span class="font-medium">AX</span>
            <span class="text-muted-color">Australia (ASX)</span>
          </div>
        </div>
        <div class="text-xs text-muted-color">
          Leave empty for US stocks (NYSE/NASDAQ)
        </div>
      </div>
    </div>
  </Popover>

  <BaseForm
    v-if="!loading"
    class="flex flex-col gap-4 p-1"
    :disabled="isReadOnly"
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
              record.investment_type === 'crypto'
                ? 'pi-bitcoin'
                : 'pi-chart-line',
            ]"
          />
        </div>
        <div class="flex flex-col gap-1 min-w-0">
          <span class="text-lg font-medium tracking-tight text-ink truncate">
            {{ record.name }}
          </span>
          <div class="flex flex-row flex-wrap items-center gap-2 text-xs">
            <span
              class="rounded-full px-2 py-0.5 font-medium bg-sunken text-ink"
            >
              {{ record.ticker }}
            </span>
            <span
              class="rounded-full px-2 py-0.5 font-medium bg-sunken text-muted"
            >
              {{ investmentTypeLabel }}
            </span>
            <span class="text-faint">{{ record.currency }}</span>
            <span v-if="record.account" class="text-muted">
              · {{ record.account.name }}
            </span>
          </div>
        </div>
        <div class="flex flex-row items-center gap-1 ml-auto">
          <button
            v-tooltip="'Fetch latest price and recalculate PnL'"
            type="button"
            class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Sync asset price"
            @click="syncAssetPrice(record?.id!)"
          >
            <i class="pi pi-sync text-xs" />
          </button>
          <button
            v-tooltip="'Recalculate PnL for all assets in this account'"
            type="button"
            class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Sync account assets"
            @click="syncAssetAccountBalance(record?.account?.id!)"
          >
            <i class="pi pi-wallet text-xs" />
          </button>
        </div>
      </div>

      <span class="text-xs text-muted">
        Mostly read only due to re-calculations. To change it, delete the asset
        and create a new one. That also deletes all related trades and reverses
        their effects.
      </span>

      <div v-if="hasHoldings" class="grid grid-cols-2 gap-2">
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
          <span class="label">Current value</span>
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

      <div v-if="hasHoldings" id="asset-tiles" class="grid grid-cols-3 gap-2">
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
        v-if="record.id && hasHoldings"
        class="flex flex-col gap-3 rounded-2xl border border-line p-4"
      >
        <div class="flex flex-col gap-1">
          <span class="label">Market value</span>
          <span class="text-xs text-muted">
            Total market value over time, including new purchases. P&L shows
            gain or loss against the all-time cost basis.
          </span>
        </div>
        <InvestmentAssetWidget :asset-id="record.id" :chart-height="200" />
        <div class="flex flex-row gap-4 items-center text-xs text-muted">
          <span><small>—</small> Market value</span>
          <span><small>· · ·</small> Cost basis</span>
        </div>
      </section>

      <section
        v-if="record.id && hasHoldings"
        class="flex flex-col gap-3 rounded-2xl border border-line p-4"
      >
        <div class="flex flex-row justify-between items-center">
          <span class="label">Income</span>
          <Button
            v-if="hasPermission('manage_data')"
            :label="showIncomeForm ? 'Cancel' : 'Add income'"
            size="small"
            :class="showIncomeForm ? 'delete-button' : 'main-button'"
            @click="showIncomeForm = !showIncomeForm"
          />
        </div>
        <span class="text-xs text-muted">
          Dividends and interest for stocks and ETFs as cash amounts. Staking
          rewards for crypto as a quantity. Rewards add to your holdings; cash
          income does not.
        </span>
        <InvestmentIncomeForm
          v-if="showIncomeForm"
          :asset-id="record.id!"
          :asset-currency="record.currency"
          :investment-type="record.investment_type"
          @complete-operation="onIncomeAdded"
        />
        <InvestmentIncomePaginated
          ref="incomeListRef"
          :asset-id="record.id!"
          :asset-currency="record.currency"
          :investment-type="record.investment_type"
        />
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Details</span>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="r$.record.name.$errors[0]"
          >
            <label>Name</label>
          </ValidationError>
          <InputText
            v-model="record.name"
            size="small"
            placeholder="Input asset name"
          />
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div
            v-for="tile in detailTiles"
            :key="tile.label"
            class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0"
          >
            <span class="text-xs text-muted">{{ tile.label }}</span>
            <span class="text-sm font-medium text-ink truncate">
              {{ tile.value || "-" }}
            </span>
          </div>
        </div>
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Auditing</span>
        <AuditTrail
          :record-id="props.recordId!"
          :events="['create', 'update']"
          :categories="['investment_asset']"
        />
      </section>
    </template>

    <template v-else>
      <div class="flex flex-row w-full justify-center">
        <SelectButton
          v-model="selectedInvestmentType"
          size="small"
          class="text-sm"
          :options="investmentTypes"
          :allow-empty="false"
        />
      </div>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <span class="label">Holding</span>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="r$.record.account.$errors.$self?.[0]"
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
            @complete="searchAccount"
          />
        </div>
        <div class="flex flex-col gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="r$.record.name.$errors[0]"
          >
            <label>Name</label>
          </ValidationError>
          <InputText
            v-model="record.name"
            size="small"
            placeholder="Input asset name"
          />
        </div>
      </section>

      <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
        <div class="flex flex-row items-center justify-between">
          <span class="label">Ticker</span>
          <button
            v-tooltip="'Formatting guide'"
            type="button"
            class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Formatting guide"
            @click="toggleInfoPopup"
          >
            <i class="pi pi-info-circle text-xs" />
          </button>
        </div>
        <div class="flex flex-row w-full gap-4">
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="r$.tickerData.name.$errors[0]"
            >
              <label>Ticker</label>
            </ValidationError>
            <InputText
              v-model="tickerData.name"
              size="small"
              placeholder="Input ticker"
            />
          </div>
          <div class="flex flex-col gap-1 w-6/12">
            <ValidationError
              :is-required="true"
              :message="r$.record.currency.$errors[0]"
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
        <div
          v-if="selectedInvestmentType.toLowerCase() !== 'crypto'"
          class="flex flex-col gap-1 w-full"
        >
          <ValidationError
            :is-required="false"
            :message="r$.tickerData.exchange?.$errors[0]"
          >
            <label>Exchange</label>
          </ValidationError>
          <InputText
            v-model="tickerData.exchange"
            size="small"
            placeholder="Input exchange"
          />
        </div>
      </section>
    </template>
  </BaseForm>
  <ShowLoading v-else :num-fields="5" />

  <div class="flex flex-col mt-4 gap-4">
    <Button
      class="main-button"
      :label="(mode == 'create' ? 'Insert' : 'Update') + ' asset'"
      :disabled="loading"
      @click="manageRecord"
    />
    <Button
      v-if="mode == 'update'"
      label="Delete asset"
      class="delete-button"
      :disabled="loading"
      @click="deleteConfirmation(record.id!)"
    />
  </div>
</template>

<style scoped>
@media (max-width: 640px) {
  #asset-tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  #asset-tiles > :last-child:nth-child(odd) {
    grid-column: 1 / -1;
    justify-self: center;
    width: calc(50% - 0.25rem);
  }
}
</style>
