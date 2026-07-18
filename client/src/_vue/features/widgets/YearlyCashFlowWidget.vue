<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import type { Account } from "../../../models/account_models.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import YearlyCashFlowBreakdownChart from "../../components/charts/YearlyCashFlowBreakdownChart.vue";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import type { YearlyCashFlowResponse } from "../../../models/analytics_models.ts";

withDefaults(
  defineProps<{
    isMobile?: boolean;
  }>(),
  {
    isMobile: false,
  },
);

const analyticsStore = useAnalyticsStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const cashFlow = ref<YearlyCashFlowResponse>({ year: 0, months: [] });
const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(null);
const selectedSeries = ref<string | null>(null);

const seriesOptions = [
  { label: "Inflows", value: "Inflows" },
  { label: "Outflows", value: "Outflows" },
  { label: "Investments", value: "Investments" },
  { label: "Savings", value: "Savings" },
  { label: "Debt Repayments", value: "Debt Repayments" },
];

const isLoadingStats = ref(false);

async function getData(year: number | null, account: number | null = null) {
  isLoadingStats.value = true;
  // Clear data first to prevent chart rendering with stale data
  cashFlow.value = { year: 0, months: [] };

  if (!year) {
    year = new Date().getFullYear();
  }

  try {
    const params: any = { year: year };
    if (account) {
      params.account = account;
    }

    cashFlow.value =
      await analyticsStore.getYearlyCashFlowOverviewForYear(params);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    isLoadingStats.value = false;
  }
}

async function loadYears() {
  try {
    const result = await analyticsStore.getAvailableStatsYears(null);

    years.value = Array.isArray(result) ? result.map((y) => y.year) : [];

    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
      ? current
      : (years.value[0] ?? current);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function loadAccounts() {
  try {
    accounts.value = await accStore.getAccountsBySubtype("checking");
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

onMounted(async () => {
  await loadAccounts();
  await loadYears();
  const defaultChecking = accounts.value.find(
    (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
  );
  if (defaultChecking) {
    selectedAccountID.value = defaultChecking.id;
  }
  await getData(null, selectedAccountID.value);
});

watch(
  () => [selectedYear.value, selectedAccountID.value] as const,
  async ([year, account]) => {
    await getData(year, account);
  },
  { flush: "post" },
);
</script>

<template>
  <div class="flex flex-col w-full p-2 gap-4">
    <div
      v-if="years.length > 0"
      class="flex flex-row gap-2 w-full justify-between items-center"
    >
      <div class="mobile-hide flex flex-col gap-1">
        <span class="text-sm" style="color: var(--text-secondary)">
          Select a year, account, and cash flow category to filter the chart.
        </span>
      </div>

      <div id="selects-row" class="flex flex-row flex-wrap gap-2 justify-end">
        <Select
          v-model="selectedYear"
          size="small"
          style="width: 150px"
          :options="years"
        />
        <Select
          v-model="selectedAccountID"
          size="small"
          style="width: 150px"
          :options="accounts"
          option-value="id"
          placeholder="All accounts"
          show-clear
        >
          <template #value="slotProps">
            <span v-if="slotProps.value">
              {{ accounts.find((a) => a.id === slotProps.value)?.name }}
            </span>
            <span v-else>All accounts</span>
          </template>
          <template #option="slotProps">
            <div class="flex flex-col">
              <span class="font-semibold">{{ slotProps.option.name }}</span>
              <span class="text-xs" style="color: var(--text-secondary)">
                {{
                  vueHelper.formatString(
                    slotProps.option.account_type?.sub_type,
                  )
                }}
              </span>
            </div>
          </template>
        </Select>
        <Select
          v-model="selectedSeries"
          size="small"
          style="width: 150px"
          :options="seriesOptions"
          option-label="label"
          option-value="value"
          placeholder="All"
          show-clear
        />
      </div>
    </div>

    <ShowLoading v-if="isLoadingStats" :num-fields="7" />
    <YearlyCashFlowBreakdownChart
      v-else-if="cashFlow.months.length > 0"
      :key="`chart-${selectedYear}-${selectedAccountID ?? 'all'}-${cashFlow.months.length}`"
      :is-mobile="isMobile"
      :data="cashFlow"
      :selected-series="selectedSeries"
    />
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #selects-row {
    width: 100%;
  }

  #selects-row > * {
    flex: 1 1 calc(50% - 4px);
    width: auto !important;
  }
}
</style>
