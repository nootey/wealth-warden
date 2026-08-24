<script setup lang="ts">
import { onMounted, ref } from "vue";
import ShowLoading from "./base/ShowLoading.vue";
import { useInvestmentStore } from "../../services/stores/investment_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useChartColors } from "../../style/theme/chartColors.ts";
import vueHelper from "../../utils/vue_helper.ts";
import type {
  PortfolioReturns,
  PortfolioReturnRow,
} from "../../models/investment_models.ts";

const investmentStore = useInvestmentStore();
const toastStore = useToastStore();
const { colors } = useChartColors();

const loading = ref(false);
const returns = ref<PortfolioReturns | null>(null);

function rateColor(row: PortfolioReturnRow): string {
  if (row.rate === null) return "var(--text-secondary)";
  return Number(row.rate) >= 0 ? colors.value.pos : colors.value.neg;
}

function rateText(row: PortfolioReturnRow): string {
  if (row.rate === null) return "-";
  const pct = vueHelper.displayAsPercentage(row.rate);
  return Number(row.rate) >= 0 ? `+${pct}` : String(pct);
}

onMounted(async () => {
  await loadReturns();
});

async function loadReturns(): Promise<void> {
  loading.value = true;
  try {
    returns.value = await investmentStore.getReturns();
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div
    id="returns-panel"
    class="flex flex-col w-full p-4 gap-5 rounded-2xl"
    style="
      background-color: var(--background-secondary);
      border: 1px solid var(--border-color);
    "
  >
    <div class="flex flex-col gap-1">
      <span class="font-bold">Return</span>
      <span class="text-sm" style="color: var(--text-secondary)">
        Money-weighted return per year (XIRR). The rate is measured
        in USD, but the values below are displayed in {{ returns?.currency ?? "native" }}
        terms.
      </span>
    </div>

    <ShowLoading v-if="loading" :num-fields="4" />

    <template v-else-if="returns">
      <div
        v-if="returns.unpriced_assets > 0"
        class="flex flex-row items-center gap-2 p-2 rounded-lg text-sm"
        style="
          border: 1px solid var(--border-color);
          color: var(--text-secondary);
        "
      >
        <i class="pi pi-exclamation-triangle" />
        <span>
          {{ returns.unpriced_assets }}
          {{ returns.unpriced_assets === 1 ? "holding has" : "holdings have" }}
          no price yet, so they are left out of this return.
        </span>
      </div>

      <div
        id="returns-headline"
        class="flex flex-row flex-wrap gap-3 p-4 rounded-xl items-center"
        style="border: 1px solid var(--border-color)"
      >
        <span
          class="text-xl font-bold"
          :style="{ color: rateColor(returns.portfolio) }"
        >
          {{ rateText(returns.portfolio) }}
        </span>
        <span class="text-sm" style="color: var(--text-secondary)">
          <template v-if="returns.portfolio.rate !== null">
            per year, across
            <b>
              {{
                vueHelper.displayAsCurrency(
                  returns.portfolio.current_value,
                  returns.currency,
                )
              }}
            </b>
            of holdings
          </template>
          <template v-else>
            Portfolio rate unavailable: {{ returns.portfolio.reason }}.
          </template>
        </span>
      </div>

      <span
        v-if="returns.assets.length === 0"
        class="text-sm"
        style="color: var(--text-secondary)"
      >
        No priced holdings yet - add a trade, or wait for the next price sync.
      </span>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="row in returns.assets"
          :key="row.key"
          class="flex flex-row items-center justify-between p-2 rounded-lg gap-2"
          style="border: 1px solid var(--border-color)"
        >
          <div class="flex flex-col min-w-0">
            <span class="text-sm font-semibold truncate">{{ row.key }}</span>
            <span
              class="text-xs truncate"
              style="color: var(--text-secondary)"
              >{{ row.label }}</span
            >
          </div>
          <div class="flex flex-row items-center gap-4 shrink-0">
            <b class="text-sm" style="color: var(--text-secondary)">
              {{
                vueHelper.displayAsCurrency(row.current_value, returns.currency)
              }}
            </b>
            <span
              v-if="row.rate !== null"
              class="text-sm font-semibold"
              :style="{ color: rateColor(row) }"
            >
              {{ rateText(row) }}
            </span>
            <span v-else class="text-xs" style="color: var(--text-secondary)">
              {{ row.reason }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
