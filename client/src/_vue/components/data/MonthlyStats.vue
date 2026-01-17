<script setup lang="ts">
import { useStatisticsStore } from "../../../services/stores/statistics_store.ts";
import { computed, onMounted, ref } from "vue";
import type { MonthlyStats } from "../../../models/statistics_models.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import ComparativePieChart from "../charts/ComparativePieChart.vue";

const statsStore = useStatisticsStore();
const toastStore = useToastStore();

const loading = ref(false);

const monthlyStats = ref<MonthlyStats | null>(null);

onMounted(async () => {
  await loadStats();
});

async function loadStats() {
  try {
    loading.value = true;
    const result = await statsStore.getCurrentMonthsStats(null);

    if (!result) {
      monthlyStats.value = null;
    } else {
      monthlyStats.value = result;
    }
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

// Pie chart data
const outflowLabels = computed<string[]>(() => {
  if (!monthlyStats.value?.categories?.length) return [];
  return monthlyStats.value.categories.map(
    (c) => c.category_name ?? "Uncategorized",
  );
});

const outflowValues = computed<number[]>(() => {
  if (!monthlyStats.value?.categories?.length) return [];
  return monthlyStats.value.categories.map((c) => parseFloat(c.outflow));
});

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
    <span class="text-sm" style="color: var(--text-secondary)"
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
        <h4 class="mobile-hide">Expense Breakdown</h4>
        <span class="text-sm" style="color: var(--text-secondary)"
          >View what you've spent your money on this month.</span
        >
        <div
          v-if="hasOutflowData"
          class="flex flex-column justify-content-center align-items-center"
        >
          <ComparativePieChart
            class="mt-3"
            :size="225"
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
