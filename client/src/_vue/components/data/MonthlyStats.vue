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
import { useChartColors } from "../../../style/theme/chartColors.ts";

const analyticsStore = useAnalyticsStore();
const toastStore = useToastStore();
const { colors } = useChartColors();

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

const allocations = computed(() => {
  const s = monthlyStats.value;
  if (!s) return [];
  return [
    {
      key: "savings",
      label: "Savings",
      amount: s.savings,
      rate: Number(s.savings_rate),
      color: colors.value.flow.savings,
    },
    {
      key: "investments",
      label: "Investments",
      amount: s.investments,
      rate: Number(s.investments_rate),
      color: colors.value.flow.investments,
    },
    {
      key: "debt",
      label: "Debt repayments",
      amount: s.debt_repayments,
      rate: Number(s.debt_repayment_rate),
      color: colors.value.flow.debt,
    },
  ];
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
  <section
    class="flex flex-col gap-4 rounded-2xl border border-line bg-card p-5 shadow-[var(--shadow-card)]"
  >
    <div class="flex flex-col gap-3">
      <div class="flex flex-col gap-1">
        <span class="label">This month</span>
        <span class="text-xs text-faint mobile-hide">
          Computed for all checking accounts, which are treated as main
          accounts.
        </span>
      </div>
      <div class="grid grid-cols-2 gap-2">
        <Select
          v-model="selectedYear"
          :options="yearOptions"
          option-label="label"
          option-value="value"
          size="small"
          placeholder="Year"
        />
        <Select
          v-model="selectedMonth"
          :options="monthOptions"
          option-label="label"
          option-value="value"
          size="small"
          placeholder="Month"
          :disabled="!monthOptions.length"
        />
      </div>
    </div>

    <ShowLoading v-if="loading" :num-fields="7" />

    <template v-else-if="monthlyStats">
      <div
        class="flex items-center justify-between gap-3 rounded-xl bg-sunken px-4 py-3"
      >
        <div class="flex flex-col gap-0.5">
          <span class="text-xs text-muted">Take home</span>
          <span
            class="text-xl leading-tight font-medium tracking-tight text-ink"
          >
            {{
              vueHelper.displayAsCurrency(
                monthlyStats.take_home,
                monthlyStats.currency,
              )
            }}
          </span>
        </div>
        <div class="flex flex-col items-end gap-0.5">
          <span class="text-xs text-muted">Overflow</span>
          <span class="text-sm font-medium text-ink">
            {{
              vueHelper.displayAsCurrency(
                monthlyStats.overflow,
                monthlyStats.currency,
              )
            }}
          </span>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2">
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
          <span class="text-xs text-muted">Inflows</span>
          <span class="text-base font-medium tracking-tight text-gain">
            {{
              vueHelper.displayAsCurrency(
                monthlyStats.inflow,
                monthlyStats.currency,
              )
            }}
          </span>
        </div>
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
          <span class="text-xs text-muted">Outflows</span>
          <span class="text-base font-medium tracking-tight text-loss">
            {{
              vueHelper.displayAsCurrency(
                monthlyStats.outflow,
                monthlyStats.currency,
              )
            }}
          </span>
        </div>
      </div>

      <div class="flex flex-col gap-3">
        <span class="label">Allocation rates</span>
        <div
          v-for="a in allocations"
          :key="a.key"
          class="flex flex-col gap-1.5"
        >
          <div class="flex items-center justify-between gap-2 text-sm">
            <div class="flex items-center gap-2 text-muted">
              <span
                class="inline-block w-2 h-2 rounded-full"
                :style="{ backgroundColor: a.color }"
              />
              <span>{{ a.label }}</span>
            </div>
            <div class="flex items-baseline gap-2">
              <span class="font-medium text-ink">
                {{
                  vueHelper.displayAsCurrency(a.amount, monthlyStats.currency)
                }}
              </span>
              <span class="w-14 text-right text-xs text-muted">
                {{ vueHelper.displayAsPercentage(a.rate) }}
              </span>
            </div>
          </div>
          <div class="h-1.5 w-full rounded-full bg-sunken overflow-hidden">
            <div
              class="h-full rounded-full"
              :style="{
                width: Math.min(Math.max(a.rate, 0), 1) * 100 + '%',
                backgroundColor: a.color,
              }"
            />
          </div>
        </div>
      </div>

      <div class="h-px bg-line" />

      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <span class="label">Expense breakdown</span>
          <span class="text-xs text-faint">
            What you spent your money on this month.
          </span>
        </div>
        <div v-if="hasOutflowData" class="flex justify-center py-2">
          <ComparativePieChart
            :size="260"
            :show-legend="false"
            :options="pieOptions"
            :values="outflowValues"
            :labels="outflowLabels"
          />
        </div>
        <div
          v-else
          class="flex items-center justify-center rounded-xl border border-dashed border-line p-6"
        >
          <span class="text-sm text-muted">
            No expenses found for this month.
          </span>
        </div>
      </div>
    </template>

    <span v-else class="text-sm text-muted">
      No checking accounts are currently available.
    </span>
  </section>
</template>
