<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import type {
  AvailableStatsYear,
  MonthlyStats,
} from "../../../models/analytics_models.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import ComparativePieChart from "../charts/ComparativePieChart.vue";

const analyticsStore = useAnalyticsStore();
const toastStore = useToastStore();

const loading = ref(false);
const monthlyStats = ref<MonthlyStats | null>(null);

const now = new Date();
const selectedYear = ref<number>(now.getFullYear());
const selectedMonth = ref<number>(now.getMonth() + 1);
const availableYears = ref<AvailableStatsYear[]>([]);

const yearOptions = computed(() =>
  availableYears.value.map((y) => ({ label: String(y.year), value: y.year })),
);

const monthOptions = computed(() => {
  const entry = availableYears.value.find((y) => y.year === selectedYear.value);
  if (!entry?.months?.length) return [];
  return entry.months.map((m) => ({
    label: new Date(selectedYear.value, m - 1).toLocaleString("default", {
      month: "long",
    }),
    value: m,
  }));
});

onMounted(async () => {
  await loadAvailableYears();
  await loadStats();
});

async function loadAvailableYears() {
  try {
    const result = await analyticsStore.getAvailableStatsYears(null, true);
    availableYears.value = result;

    // Default to current year if available, otherwise latest
    const currentYearEntry = result.find((y) => y.year === now.getFullYear());
    const entry = currentYearEntry ?? result[result.length - 1];
    if (!entry) return;

    selectedYear.value = entry.year;

    // Default to current month if available, otherwise latest valid month
    const months = entry.months ?? [];
    const currentMonth = now.getMonth() + 1;
    selectedMonth.value = months.includes(currentMonth)
      ? currentMonth
      : (months[months.length - 1] ?? currentMonth);
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function loadStats() {
  try {
    loading.value = true;
    const result = await analyticsStore.getCurrentMonthsStats(
      null,
      selectedYear.value,
      selectedMonth.value,
    );
    monthlyStats.value = result ?? null;
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

// When year changes, reset month to latest valid for that year
watch(selectedYear, (newYear) => {
  const entry = availableYears.value.find((y) => y.year === newYear);
  const months = entry?.months ?? [];
  const currentMonth = now.getMonth() + 1;
  selectedMonth.value = months.includes(currentMonth)
    ? currentMonth
    : (months[months.length - 1] ?? currentMonth);
});

watch(selectedMonth, async () => {
  await loadStats();
});

// Pie chart data
const MAX_SLICES = 12;

const processedOutflowData = computed(() => {
  if (!monthlyStats.value?.categories?.length)
    return { labels: [] as string[], values: [] as number[] };

  const items = monthlyStats.value.categories.map((c) => ({
    label: c.category_name ?? "Uncategorized",
    value: parseFloat(c.outflow),
  }));

  items.sort((a, b) => b.value - a.value);

  if (items.length <= MAX_SLICES)
    return {
      labels: items.map((c) => c.label),
      values: items.map((c) => c.value),
    };

  const main = items.slice(0, MAX_SLICES - 1);
  const rest = items.slice(MAX_SLICES - 1);
  main.push({
    label: "Other",
    value: rest.reduce((sum, c) => sum + c.value, 0),
  });

  return { labels: main.map((c) => c.label), values: main.map((c) => c.value) };
});

const outflowLabels = computed<string[]>(
  () => processedOutflowData.value.labels,
);
const outflowValues = computed<number[]>(
  () => processedOutflowData.value.values,
);

const hasOutflowData = computed(() => outflowValues.value?.length > 0);

const pieOptions = computed(() => ({
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: any) => {
          const label = ctx.label ?? "";
          const value = Number(ctx.parsed);
          const data = (ctx.dataset?.data ?? []) as number[];
          const total = data.reduce((a, b) => a + b, 0);
          const pct = total ? value / total : 0;

          return `${label}: ${vueHelper.displayAsCurrency(value)} · ${vueHelper.displayAsPercentage(pct)}`;
        },
      },
    },
  },
}));
</script>

<template>
  <div v-if="!loading" class="flex flex-column p-2 gap-2">
    <span class="text-sm mobile-hide" style="color: var(--text-secondary)"
      >Monthly stats are computed for all checking accounts, which are treated
      as main accounts.</span
    >

    <h4 class="mt-2 mobile-hide">Details</h4>
    <div v-if="monthlyStats" class="flex flex-column">
      <div class="flex flex-column w-full gap-2">
        <div class="flex flex-row gap-2 align-items-center">
          <span>Inflows:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats?.inflow!,
                monthlyStats.currency,
              )
            }}</b></span
          >
        </div>
        <div class="flex flex-row gap-2 align-items-center">
          <span>Outflows:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats?.outflow!,
                monthlyStats.currency,
              )
            }}</b></span
          >
        </div>
        <div class="flex flex-row gap-2 align-items-center">
          <span>Take home:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats?.take_home!,
                monthlyStats.currency,
              )
            }}</b></span
          >
        </div>
        <div class="flex flex-row gap-2 align-items-center">
          <span>Overflow:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats?.overflow!,
                monthlyStats.currency,
              )
            }}</b></span
          >
        </div>
      </div>

      <div class="flex flex-column w-full gap-2 mt-3">
        <h4 class="mobile-hide">Rates</h4>

        <div class="flex flex-row gap-2 align-items-center">
          <span>Savings:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats.savings,
                monthlyStats.currency,
              )
            }}</b></span
          >
          <span>Rate:</span>
          <span
            ><b>{{
              vueHelper.displayAsPercentage(monthlyStats.savings_rate)
            }}</b></span
          >
        </div>

        <div class="flex flex-row gap-2 align-items-center">
          <span>Investments</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats.investments,
                monthlyStats.currency,
              )
            }}</b></span
          >
          <span>Rate</span>
          <span
            ><b>{{
              vueHelper.displayAsPercentage(monthlyStats.investments_rate)
            }}</b></span
          >
        </div>

        <div class="flex flex-row gap-2 align-items-center">
          <span>Debt repayments</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(
                monthlyStats.debt_repayments,
                monthlyStats.currency,
              )
            }}</b></span
          >
          <span>Rate</span>
          <span
            ><b>{{
              vueHelper.displayAsPercentage(monthlyStats.debt_repayment_rate)
            }}</b></span
          >
        </div>
      </div>

      <div class="flex flex-column w-full gap-2 mt-3">
        <h4>Expense Breakdown</h4>
        <span class="text-sm" style="color: var(--text-secondary)"
          >View what you've spent your money on per month.</span
        >
        <div class="flex flex-row gap-2 align-items-center w-full pr-2">
          <Select
            v-model="selectedYear"
            class="w-full"
            :options="yearOptions"
            option-label="label"
            option-value="value"
            size="small"
            placeholder="Year"
          />
          <Select
            v-model="selectedMonth"
            class="w-full"
            :options="monthOptions"
            option-label="label"
            option-value="value"
            size="small"
            placeholder="Month"
            :disabled="!monthOptions.length"
          />
        </div>
        <div
          v-if="hasOutflowData"
          class="flex flex-column justify-content-center align-items-center"
        >
          <ComparativePieChart
            class="mt-3"
            :size="300"
            :show-legend="false"
            :options="pieOptions"
            :values="outflowValues"
            :labels="outflowLabels"
          />
        </div>
        <div
          v-else
          class="flex flex-column align-items-center justify-content-center p-3"
          style="border: 1px dashed var(--border-color); border-radius: 16px"
        >
          <span class="text-sm" style="color: var(--text-secondary)">
            No expenses found for this month.
          </span>
        </div>
      </div>
    </div>
    <div v-else>
      <span>No checking accounts are currently available.</span>
    </div>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped></style>
