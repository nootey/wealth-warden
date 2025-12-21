<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useConfirm} from "primevue/useconfirm";
import {computed, nextTick, onMounted, ref} from "vue";
import type {Account} from "../../../models/account_models.ts";
import type {InvestmentHolding} from "../../../models/investment_models.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import {required} from "@vuelidate/validators";
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

const investmentTypes = ref<string[]>(["crypto", "stock", "etf"])

const selectedInvestmentType = ref<string>(
  investmentTypes.value.find((i) => i === "crypto") ?? "etf"
);

const availableAccounts = computed(() => {
  const typeMap: Record<string, string[]> = {
    crypto: ["wallet", "exchange"],
    stock: ["brokerage", "retirement", "pension"],
    etf: ["brokerage", "retirement", "pension", "mutual_fund"],
  };

  const allowedSubtypes = typeMap[selectedInvestmentType.value] || [];

  return accounts.value.filter(acc =>
    allowedSubtypes.includes(acc.account_type.sub_type)
  );
});

const rules = {
  record: {
    name: {
      required,
      $autoDirty: true,
    },
    ticker: {
      required,
      $autoDirty: true,
    },
    account_id: {
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
};

const v$ = useVuelidate(rules, { record });

onMounted(async () => {

  accounts.value = await accountStore.getAllAccounts(true, true);

  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

function initData(): InvestmentHolding {
  return {
    account_id: null,
    investment_type: "stock",
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
  const isValid = await v$.value.record.$validate();
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
    };

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function manageRecord() {

  const recordData = {
    account_id: record.value.account_id,
    investment_type: record.value.investment_type,
    quantity: record.value.quantity,
    name: record.value.name,
    ticker: record.value.ticker,
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
            :message="v$.record.account_id.$errors[0]?.$message"
          >
            <label>Account</label>
          </ValidationError>
          <AutoComplete
            v-model="record.account_id"
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

    </div>

  </div>
</template>

<style scoped>

</style>