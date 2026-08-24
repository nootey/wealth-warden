<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import ComparativePieChart from "./charts/ComparativePieChart.vue";
import ShowLoading from "./base/ShowLoading.vue";
import { useInvestmentStore } from "../../services/stores/investment_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { CATEGORY_PALETTE } from "../../style/theme/chartColors.ts";
import vueHelper from "../../utils/vue_helper.ts";
import type {
  AllocationGroupKey,
  AllocationRow,
  PortfolioAllocation,
} from "../../models/investment_models.ts";

const investmentStore = useInvestmentStore();
const toastStore = useToastStore();

const loading = ref(false);
const allocation = ref<PortfolioAllocation | null>(null);

const groupOptions: { label: string; value: AllocationGroupKey }[] = [
  { label: "Type", value: "type" },
  { label: "Ticker", value: "ticker" },
  { label: "Currency", value: "currency" },
  { label: "Account", value: "account" },
];
const selectedGroup = ref<AllocationGroupKey>("type");

const groupDescriptions: Record<AllocationGroupKey, string> = {
  type: "Splits the portfolio by asset class - stock, ETF or crypto.",
  ticker:
    "Splits the portfolio by holding. The same ticker in two accounts counts once.",
  currency: "Splits the portfolio by the currency each holding is priced in.",
  account: "Splits the portfolio by the account that holds the position.",
};

const rows = computed<AllocationRow[]>(
  () => allocation.value?.groups[selectedGroup.value] ?? [],
);

const chartValues = computed<number[]>(() =>
  rows.value.map((r) => Number(r.value)),
);
const chartLabels = computed<string[]>(() => rows.value.map((r) => r.label));

const chartOptions = { cutout: "62%" };

function sliceColor(index: number): string {
  return CATEGORY_PALETTE[index % CATEGORY_PALETTE.length];
}

onMounted(async () => {
  await loadAllocation();
});

async function loadAllocation(): Promise<void> {
  loading.value = true;
  try {
    allocation.value = await investmentStore.getAllocation();
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div
    id="allocation-panel"
    class="flex flex-col w-full p-4 gap-5 rounded-2xl"
    style="
      background-color: var(--background-secondary);
      border: 1px solid var(--border-color);
    "
  >
    <div class="flex flex-row flex-wrap items-center justify-between gap-3">
      <div class="flex flex-col gap-1">
        <span class="font-bold">Allocation</span>
        <span
          v-if="allocation"
          class="text-xs"
          style="color: var(--text-secondary)"
        >
          {{
            vueHelper.displayAsCurrency(
              allocation.total_value,
              allocation.currency,
            )
          }}
          invested
        </span>
      </div>
      <div id="allocation-groups" class="flex flex-row">
        <SelectButton
          v-model="selectedGroup"
          style="font-size: 0.875rem"
          size="small"
          :options="groupOptions"
          option-label="label"
          option-value="value"
          :allow-empty="false"
        />
      </div>
    </div>

    <div class="flex flex-col gap-1">
      <span
        v-if="allocation"
        class="text-sm"
        style="color: var(--text-secondary)"
      >
        Holdings priced in non-native account currency, are converted to
        {{ allocation.currency }}, at the latest known rate.
      </span>
      <span class="text-sm" style="color: var(--text-secondary)">
        {{ groupDescriptions[selectedGroup] }}
      </span>
    </div>

    <ShowLoading v-if="loading" :num-fields="4" />

    <span
      v-else-if="rows.length === 0"
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      No priced holdings yet - add a trade, or wait for the next price sync.
    </span>

    <template v-else>
      <div
        v-if="allocation && allocation.unpriced_assets > 0"
        class="flex flex-row items-center gap-2 p-2 rounded-lg text-sm"
        style="
          border: 1px solid var(--border-color);
          color: var(--text-secondary);
        "
      >
        <i class="pi pi-exclamation-triangle" />
        <span>
          {{ allocation.unpriced_assets }}
          {{
            allocation.unpriced_assets === 1 ? "holding has" : "holdings have"
          }}
          no price yet, so they are left out of these weights.
        </span>
      </div>

      <div id="allocation-body" class="flex flex-row items-center gap-6">
        <ComparativePieChart
          :size="260"
          :show-legend="false"
          :options="chartOptions"
          :values="chartValues"
          :labels="chartLabels"
        />

        <div class="flex flex-col w-full gap-2">
          <div
            v-for="(row, index) in rows"
            :key="row.key"
            class="flex flex-row items-center justify-between p-2 rounded-lg gap-2"
            style="border: 1px solid var(--border-color)"
          >
            <div class="flex flex-row items-center gap-2 min-w-0">
              <span
                class="shrink-0 rounded-full"
                :style="{
                  width: '10px',
                  height: '10px',
                  backgroundColor: sliceColor(index),
                }"
              />
              <span class="text-sm font-semibold truncate">{{
                row.label
              }}</span>
            </div>
            <div class="flex flex-row items-center gap-4 shrink-0">
              <span class="text-sm" style="color: var(--text-secondary)">
                {{
                  vueHelper.displayAsCurrency(row.value, allocation?.currency)
                }}
              </span>
              <span class="text-sm font-semibold">
                {{ vueHelper.displayAsPercentage(row.weight) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #allocation-body {
    flex-direction: column;
  }

  /* The buttons wrap onto their own line, so center them there */
  #allocation-groups {
    width: 100%;
    justify-content: center;
  }
}
</style>
