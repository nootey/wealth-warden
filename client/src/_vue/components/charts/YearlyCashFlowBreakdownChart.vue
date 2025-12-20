<script setup lang="ts">
import { computed, ref, onUnmounted, onMounted } from "vue";
import Chart from "primevue/chart";
import {
  Chart as ChartJS,
  BarController,
  BarElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
} from "chart.js";
import type { YearlyCashFlowResponse } from "../../../models/chart_models";
import vueHelper from "../../../utils/vue_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";

ChartJS.register(
  BarController,
  BarElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
);

const props = defineProps<{ data: YearlyCashFlowResponse }>();
const { colors } = useChartColors();

const labels = computed(() =>
  props.data.months.map((m) =>
    new Date(props.data.year, m.month - 1).toLocaleString("default", {
      month: "short",
    }),
  ),
);

const inflowsArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.inflows as any)),
);

const outflowsArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.outflows as any)),
);

const investmentsArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.investments as any)),
);

const savingsArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.savings as any)),
);

const debtArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.debt_repayments as any)),
);

const takeHomeArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.take_home as any)),
);

const overflowArr = computed(() =>
  props.data.months.map((m) => toNumber(m.categories.overflow ?? (0 as any))),
);

const hasAnyData = computed(() => {
  const totalIn = inflowsArr.value.reduce((a, b) => a + b, 0);
  const totalOut = outflowsArr.value.reduce((a, b) => a + b, 0);
  return totalIn !== 0 || totalOut !== 0;
});

const chartData = computed(() => ({
  labels: labels.value,
  datasets: [
    {
      label: "Inflows",
      data: inflowsArr.value,
      backgroundColor: "#3b82f6",
      stack: "stack0",
    },
    {
      label: "Outflows",
      data: outflowsArr.value.map((v) => Math.abs(v)),
      backgroundColor: "#8b5cf6",
      stack: "stack0",
    },
    {
      label: "Investments",
      data: investmentsArr.value,
      backgroundColor: "#ec4899",
      stack: "stack0",
    },
    {
      label: "Savings",
      data: savingsArr.value,
      backgroundColor: "#eab308",
      stack: "stack0",
    },
    {
      label: "Debt Repayments",
      data: debtArr.value,
      backgroundColor: "#06b6d4",
      stack: "stack0",
    },
    // Display only overflow on the chart, so that we can see when we went into negative for the month
    {
      label: "Overflow",
      data: overflowArr.value.map((v) => -Math.abs(v)),
      backgroundColor: colors.value.neg,
      stack: "stack0",
    },
  ],
}));

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: "index", intersect: false },
  events: ["mousemove", "mouseout", "touchstart", "touchmove"],

  plugins: {
    legend: {
      position: "top",
      labels: {
        color: colors.value.axisText,
        usePointStyle: true,
        pointStyle: "circle",
        boxWidth: 8,
        boxHeight: 8,
        padding: 16,
      },
    },
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
      footerFont: { weight: "600", size: 12 },
      footerSpacing: 1,

      callbacks: {
        title: (items: any[]) => items?.[0]?.label ?? "",
        label: (ctx: any) => {
          if (
            ctx.dataset.label === "Take Home" ||
            ctx.dataset.label === "Overflow"
          ) {
            return null;
          }
          return `${ctx.dataset.label}: ${vueHelper.displayAsCurrency(Math.abs(ctx.parsed?.y))}`;
        },
        footer: (items: any[]) => {
          if (!items?.length) return "";
          const i = items[0].dataIndex;
          const takeHome = takeHomeArr.value[i] ?? 0;
          const overflow = overflowArr.value[i] ?? 0;

          const lines = [];

          if (takeHome > 0) {
            lines.push(`Take Home: ${vueHelper.displayAsCurrency(takeHome)}`);
          }

          if (overflow < 0) {
            lines.push(
              `Overflow: ${vueHelper.displayAsCurrency(Math.abs(overflow))}`,
            );
          }

          return lines.join("\n");
        },
      },
      footerColor: (_ctx: any) => {
        const it = _ctx?.tooltip?.dataPoints?.[0];
        if (!it) return colors.value.axisText;
        const i = it.dataIndex;
        const takeHome = takeHomeArr.value[i] ?? 0;
        return takeHome > 0 ? colors.value.pos : colors.value.neg;
      },
    },
  },

  scales: {
    x: {
      type: "category",
      stacked: true,
      grid: { display: false, drawBorder: false },
      ticks: { color: colors.value.axisText, maxRotation: 0, minRotation: 0 },
      border: { color: colors.value.axisBorder },
    },
    y: {
      stacked: true,
      grid: { display: false, drawBorder: false },
      ticks: {
        display: true,
        color: colors.value.axisText,
        callback: (v: number) => vueHelper.displayAsCurrency(v),
      },
      border: { color: colors.value.axisBorder },
    },
  },
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

function toNumber(v: string | string[] | undefined): number {
  if (Array.isArray(v)) return v.reduce((a, s) => a + (parseFloat(s) || 0), 0);
  if (typeof v === "string") return parseFloat(v) || 0;
  return 0;
}
</script>

<template>
  <Chart
    v-if="hasAnyData && isChartReady"
    ref="chartRef"
    type="bar"
    :data="chartData"
    :options="chartOptions"
    style="width: 100%; height: 400px"
  />
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
