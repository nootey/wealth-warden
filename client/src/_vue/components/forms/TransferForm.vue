<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { Account } from "../../../models/account_models.ts";
import type { Transfer } from "../../../models/transaction_models.ts";
import { useRegle } from "@regle/core";
import { required } from "@regle/rules";
import ValidationError from "../validation/ValidationError.vue";
import {
  decimalMax,
  decimalMin,
  decimalValid,
} from "../../../validators/currency.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import ShowLoading from "../base/ShowLoading.vue";
import dayjs from "dayjs";

const props = defineProps<{
  accounts: Account[];
  transfer: Transfer;
  mode?: "create" | "update";
}>();

const emit = defineEmits<{
  (event: "update:transfer", value: Transfer): void;
}>();

const settingsStore = useSettingsStore();
const loading = ref(false);

const localTransfer = ref<{
  source: Account | null;
  destination: Account | null;
  amount: string | null;
  notes: string | null;
  created_at: Date | null;
}>({
  source:
    props.mode === "update" ? (props.transfer.from?.account ?? null) : null,
  destination:
    props.mode === "update" ? (props.transfer.to?.account ?? null) : null,
  amount: props.mode === "update" ? (props.transfer.amount ?? null) : null,
  notes: props.transfer.notes ?? null,
  created_at: props.transfer.created_at
    ? dayjs(props.transfer.created_at).toDate()
    : dayjs().toDate(),
});

const rules = {
  source: { $self: { required } },
  destination: { $self: { required } },
  amount: {
    required,
    decimalValid,
    decimalMin: decimalMin(0),
    decimalMax: decimalMax(1_000_000_000),
  },
  created_at: {
    required,
  },
  notes: {},
};

const { r$ } = useRegle(localTransfer, rules);
const amountRef = computed({
  get: () => localTransfer.value.amount,
  set: (v) => (localTransfer.value.amount = v),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

watch(
  localTransfer,
  (val) => {
    emit("update:transfer", {
      ...props.transfer,
      source_id: val.source?.id ?? null,
      destination_id: val.destination?.id ?? null,
      amount: val.amount ?? null,
      notes: val.notes ?? null,
      created_at: val.created_at ?? null,
    });
  },
  { deep: true },
);

const filteredSourceAccounts = ref<Account[]>([]);
const filteredDestinationAccounts = ref<Account[]>([]);

function searchAccount(
  type: "source" | "destination",
  event: { query: string },
) {
  setTimeout(() => {
    let results = props.accounts;

    if (event.query?.trim().length) {
      results = results.filter((a) =>
        a.name.toLowerCase().startsWith(event.query.toLowerCase()),
      );
    }

    if (type === "source") {
      // exclude currently selected destination
      if (localTransfer.value.destination) {
        results = results.filter(
          (a) => a.id !== localTransfer.value?.destination?.id,
        );
      }
      filteredSourceAccounts.value = results;
    } else {
      // exclude currently selected source
      if (localTransfer.value.source) {
        results = results.filter(
          (a) => a.id !== localTransfer.value?.source?.id,
        );
      }
      filteredDestinationAccounts.value = results;
    }
  }, 200);
}

defineExpose({ r$, localTransfer });
</script>

<template>
  <div v-if="!loading" class="flex flex-col gap-4">
    <div class="flex flex-row w-full">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="r$.source.$errors.$self?.[0]"
        >
          <label>Source</label>
        </ValidationError>
        <AutoComplete
          v-model="localTransfer.source"
          size="small"
          :suggestions="filteredSourceAccounts"
          option-label="name"
          placeholder="Select source account"
          dropdown
          :disabled="mode === 'update'"
          :readonly="mode === 'update'"
          @complete="(e) => searchAccount('source', e)"
        />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="r$.destination.$errors.$self?.[0]"
        >
          <label>Destination</label>
        </ValidationError>
        <AutoComplete
          v-model="localTransfer.destination"
          size="small"
          :suggestions="filteredDestinationAccounts"
          option-label="name"
          placeholder="Select destination account"
          dropdown
          :disabled="mode === 'update'"
          :readonly="mode === 'update'"
          @complete="(e) => searchAccount('destination', e)"
        />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="true" :message="r$.amount.$errors[0]">
          <label>Amount</label>
        </ValidationError>
        <InputNumber
          v-model="amountNumber"
          size="small"
          mode="currency"
          :currency="settingsStore.defaultCurrency"
          :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
          :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
        />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError
          :is-required="true"
          :message="r$.created_at.$errors[0]"
        >
          <label>Date</label>
        </ValidationError>
        <DatePicker
          v-model="localTransfer.created_at"
          date-format="dd/mm/yy"
          show-icon
          fluid
          icon-display="input"
          size="small"
        />
      </div>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="false" :message="r$.notes.$errors[0]">
          <label>Notes</label>
        </ValidationError>
        <InputText
          v-model="localTransfer.notes"
          size="small"
          placeholder="Describe transfer"
        />
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="6" />
</template>
