<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {computed, nextTick, onMounted, ref} from "vue";
import type {Account} from "../../../models/account_models.ts";
import type {InvestmentHolding, TickerData} from "../../../models/investment_models.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import {required, requiredIf} from "@vuelidate/validators";
import {decimalMax, decimalMin, decimalValid} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const apiPrefix = "investments"

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const accountStore = useAccountStore();


const loading = ref(false);

const accounts = ref<Account[]>([]);
const record = ref<InvestmentHolding>(initData());
const filteredAccounts = ref<Account[]>([]);

const quantitytRef = computed({
  get: () => record.value.quantity,
  set: (v) => (record.value.quantity = v),
});
const { number: quantityNumber } = currencyHelper.useMoneyField(quantitytRef, 2);

const investmentTypes = ref<string[]>(["Crypto", "Stock", "ETF"])

const selectedInvestmentType = ref<string>(
  investmentTypes.value.find((i) => i === "Crypto") ?? "ETF"
);

const availableAccounts = computed(() => {
  const typeMap: Record<string, string[]> = {
    crypto: ["wallet", "exchange"],
    stock: ["brokerage", "retirement", "pension"],
    etf: ["brokerage", "retirement", "pension", "mutual_fund"],
  };

  const allowedSubtypes = typeMap[selectedInvestmentType.value.toLowerCase()] || [];

  return accounts.value.filter(acc =>
    allowedSubtypes.includes(acc.account_type.sub_type)
  );
});

const tickerData = ref<TickerData>({
  name: "",
  exchange: "",
  currency: ""
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
      $autoDirty: true
    },
    exchange: {
      required: requiredIf(() => selectedInvestmentType.value.toLowerCase() === 'crypto'),
      $autoDirty: true
    },
    currency: {
      required: requiredIf(() => selectedInvestmentType.value.toLowerCase() === 'crypto'),
      $autoDirty: true
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

function initData(): InvestmentHolding {
  return {
    account: null,
    investment_type: "crypto",
    name: "",
    ticker: "",
    quantity: ""
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
    const data = await sharedStore.getRecordByID(apiPrefix, id, {
      deleted: true,
    });

    // Parse ticker if crypto
    if (data.ticker && data.investment_type.toLowerCase() === "crypto" && data.ticker.includes(":")) {
      const [exchange, symbolCurrency] = data.ticker.split(":");
      const currency = symbolCurrency.slice(-4);
      const name = symbolCurrency.slice(0, -4);

      tickerData.value = { name, exchange, currency };
    } else {
      tickerData.value = { name: data.ticker || "", exchange: "", currency: "" };
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

  let ticker = tickerData.value.name;

  if (selectedInvestmentType.value.toLowerCase() === "crypto" && tickerData.value.exchange && tickerData.value.currency) {
    ticker = `${tickerData.value.exchange}:${tickerData.value.name}${tickerData.value.currency}`;
  }

  const recordData = {
    account_id: record.value.account.id,
    investment_type: record.value.investment_type.toLowerCase(),
    quantity: record.value.quantity,
    name: record.value.name,
    ticker: ticker,
  };

  try {
    let response = null;

    switch (props.mode) {
      case "create":
        response = await sharedStore.createRecord(
          apiPrefix,
          recordData,
        );
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

    <div
      class="flex flex-row w-full justify-content-center"
    >
      <div class="flex flex-column w-50">
        <SelectButton
          v-model="selectedInvestmentType"
          style="font-size: 0.875rem"
          size="small"
          :options="investmentTypes"
          :allow-empty="false"
        />
      </div>
    </div>

    <div class="flex flex-column gap-3">
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
          />
        </div>
      </div>

      <div v-if="selectedInvestmentType.toLowerCase() === 'crypto'" class="flex flex-row w-full gap-2">
        <div class="flex flex-column gap-1 w-6">
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
          />
        </div>
        <div class="flex flex-column gap-1 w-6">
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
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
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
            placeholder="0,00"
          />
        </div>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <div class="flex flex-column w-full gap-2">
          <Button
            class="main-button"
            label="Create"
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

<style scoped>

</style>