<script setup lang="ts">
import { computed, onMounted, ref, onUnmounted } from "vue";
import Chart from "primevue/chart";
import type { YearlySankeyData } from "../../../models/chart_models";
import vueHelper from "../../../utils/vue_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import { Chart as ChartJS } from "chart.js";
import { SankeyController, Flow } from "chartjs-chart-sankey";

ChartJS.register(SankeyController, Flow);

const props = defineProps<{
  data: YearlySankeyData;
}>();

const { colors } = useChartColors();

const toNum = (v: string | undefined): number => parseFloat(v || "0");

const hasAnyData = computed(() => toNum(props.data.total_income) > 0);

const totalIncome = computed(() => toNum(props.data.total_income));

const chartData = computed(() => {
  const data: any[] = [];

  const savings = toNum(props.data.savings);
  const investments = toNum(props.data.investments);
  const debtRepayments = toNum(props.data.debt_repayments);
  const expenses = Math.abs(toNum(props.data.expenses));

  // Calculate allocated amount
  const allocated = savings + investments + debtRepayments + expenses;
  const unallocated = Math.max(0, totalIncome.value - allocated);

  // Income -> Primary allocations
  if (savings > 0) {
    data.push({ from: "Total Income", to: "Savings", flow: savings });
  }
  if (investments > 0) {
    data.push({ from: "Total Income", to: "Investments", flow: investments });
  }
  if (debtRepayments > 0) {
    data.push({
      from: "Total Income",
      to: "Debt Repayments",
      flow: debtRepayments,
    });
  }
  if (expenses > 0) {
    data.push({ from: "Total Income", to: "Expenses", flow: expenses });
  }

  if (unallocated > 0) {
    data.push({ from: "Total Income", to: "Unallocated", flow: unallocated });
  }

  props.data.expense_categories.forEach((cat) => {
    const amount = Math.abs(toNum(cat.amount));
    if (amount > 0) {
      data.push({ from: "Expenses", to: cat.category_name, flow: amount });
    }
  });

  return {
    datasets: [
      {
        data,
        colorFrom: (c: any) => {
          if (!c?.raw?.from) return colors.value.neg;
          if (c.raw.from === "Total Income") return colors.value.pos;
          if (c.raw.from === "Expenses") return colors.value.neg;
          return colors.value.neg;
        },
        colorTo: (c: any) => {
          if (!c?.raw?.to) return colors.value.neg;
          if (c.raw.to === "Savings") return "#3b82f6";
          if (c.raw.to === "Investments") return "#8b5cf6";
          if (c.raw.to === "Debt Repayments") return "#f97316";
          if (c.raw.to === "Expenses") return colors.value.neg;
          if (c.raw.to === "Unallocated") return "#6b7280";
          return colors.value.neg;
        },
        borderWidth: 0,
      },
    ],
  };
});

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: colors.value.ttipBg,
      borderColor: colors.value.ttipBorder,
      borderWidth: 1,
      padding: 12,
      cornerRadius: 12,
      displayColors: false,
      titleColor: colors.value.ttipTitle,
      bodyColor: colors.value.ttipText,
      titleFont: { weight: "600", size: 12 },
      bodyFont: { weight: "600", size: 14 },
      callbacks: {
        title: (items: any[]) => {
          const raw = items[0]?.raw;
          if (!raw) return "";
          return `${raw.from} → ${raw.to}`;
        },
        label: (ctx: any) => {
          if (!ctx?.raw?.flow) return "";
          const amount = ctx.raw.flow;
          const income = toNum(props.data.total_income);
          const pct = income > 0 ? ((amount / income) * 100).toFixed(1) : "0";
          return `${vueHelper.displayAsCurrency(amount)} (${pct}%)`;
        },
      },
    },
  },
  layout: {
    padding: {
      left: 10,
      right: 10,
    },
  },
  color: colors.value.axisText,
}));

const chartRef = ref<any>(null);
const isChartReady = ref(false);

onMounted(() => {
  isChartReady.value = true;
});

onUnmounted(() => {
  if (chartRef.value?.chart) {
    chartRef.value.chart.destroy();
  }
});
</script>

<template>
  <div v-if="hasAnyData" class="flex flex-column gap-3 p-2">
    <div
      class="flex flex-row align-items-center justify-content-between p-3 border-round-lg"
      style="
        background: var(--surface-card);
        border: 1px solid var(--border-color);
      "
    >
      <div class="flex flex-column gap-1">
        <span class="text-sm" style="color: var(--text-secondary)">
          Total Income
        </span>
        <span class="font-bold" style="color: var(--text-color)">
          {{ vueHelper.displayAsCurrency(totalIncome) }}
        </span>
      </div>
    </div>

    <!-- Sankey Chart -->
    <Chart
      v-if="isChartReady"
      ref="chartRef"
      type="sankey"
      :data="chartData"
      :options="chartOptions"
      style="width: 100%; height: 600px"
    />
  </div>
  <div
    v-else
    class="flex flex-column align-items-center justify-content-center mt-3 p-3 w-6"
    style="
      border: 1px dashed var(--border-color);
      border-radius: 16px;
      margin: 0 auto;
    "
  >
    <span class="text-sm" style="color: var(--text-secondary)">
      No cash flow data available for {{ props.data.year }}.
    </span>
  </div>
</template>
