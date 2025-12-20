<script setup lang="ts">
import { computed, ref, onUnmounted, onMounted } from "vue";
import Chart from "primevue/chart";
import {
  Chart as ChartJS,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
} from "chart.js";
import type { MonthlyCashFlowResponse } from "../../../models/chart_models";
import vueHelper from "../../../utils/vue_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";

ChartJS.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
);

const props = withDefaults(
  defineProps<{
    isMobile?: boolean;
    data: MonthlyCashFlowResponse;
  }>(),
  {
    isMobile: false,
  },
);

const { colors } = useChartColors();

const labels = computed(() =>
  props.data.series.map((m) =>
    new Date(props.data.year, m.month - 1).toLocaleString("default", {
      month: "short",
    }),
  ),
);
const inflowsArr = computed(() =>
  props.data.series.map((m) => toNumber(m.inflows)),
);
const outflowsArr = computed(() =>
  props.data.series.map((m) => toNumber(m.outflows)),
);

const hasAnyData = computed(() => {
  const totalIn = inflowsArr.value.reduce((a, b) => a + b, 0);
  const totalOut = outflowsArr.value.reduce((a, b) => a + b, 0);
  return totalIn !== 0 || totalOut !== 0;
});

const hoverXByChart = new WeakMap<any, number | null>();
const hoverGuidePlugin = {
  id: "hoverGuide",
  afterEvent(chart: any, args: any) {
    let next: number | null = null;
    if (args.inChartArea) {
      const a = chart.getActiveElements?.() ?? [];
      if (a.length) next = a[0].element?.$context?.parsed?.x ?? null;
    }
    if (args.event?.type === "mouseout") next = null;
    const prev = hoverXByChart.get(chart) ?? null;
    if (prev !== next) {
      hoverXByChart.set(chart, next);
      const prevAnim = chart.options.animation;
      chart.options.animation = false;
      chart.update();
      chart.options.animation = prevAnim;
    }
  },
  afterDatasetsDraw(chart: any) {
    const hv = hoverXByChart.get(chart);
    if (hv == null) return;
    const { ctx, chartArea, scales } = chart;
    const x = scales.x.getPixelForValue(hv);
    ctx.save();
    ctx.setLineDash([4, 6]);
    ctx.lineWidth = 1;
    ctx.strokeStyle = colors.value.guide;
    ctx.beginPath();
    ctx.moveTo(x, chartArea.top);
    ctx.lineTo(x, chartArea.bottom);
    ctx.stroke();
    ctx.restore();
  },
};

const chartData = computed(() => ({
  labels: labels.value,
  datasets: [
    {
      label: "Inflows",
      data: inflowsArr.value,
      borderColor: colors.value.pos,
      borderWidth: 3,
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 4,
      pointHitRadius: 16,
      fill: false,
      segment: {
        borderColor: (ctx: any) => {
          const hv = hoverXByChart.get(ctx.chart) ?? null;
          if (hv == null) return colors.value.pos;
          const x0 = ctx.p0?.parsed?.x;
          return x0 >= hv ? colors.value.dim : colors.value.pos;
        },
      },
    },
    {
      label: "Outflows",
      data: outflowsArr.value,
      borderColor: colors.value.neg,
      borderWidth: 3,
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 4,
      pointHitRadius: 16,
      fill: false,
      segment: {
        borderColor: (ctx: any) => {
          const hv = hoverXByChart.get(ctx.chart) ?? null;
          if (hv == null) return colors.value.neg;
          const x0 = ctx.p0?.parsed?.x;
          return x0 >= hv ? colors.value.dim : colors.value.neg;
        },
      },
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
        label: (ctx: any) =>
          `${ctx.dataset.label}: ${vueHelper.displayAsCurrency(ctx.parsed?.y)}`,
        footer: (items: any[]) => {
          if (!items?.length) return "";
          const i = items[0].dataIndex;
          const inflow = inflowsArr.value[i] ?? 0;
          const outflow = outflowsArr.value[i] ?? 0;
          const net = inflow - outflow;
          const sign = net >= 0 ? "+" : "−";
          const pct =
            inflow + outflow !== 0
              ? (net / Math.max(1, Math.abs(outflow))) * 100
              : null;
          const pctStr = pct == null ? "" : ` (${Math.abs(pct).toFixed(1)}%)`;
          return `Difference: ${sign} ${vueHelper.displayAsCurrency(Math.abs(net))}${pctStr}`;
        },
      },
      footerColor: (_ctx: any) => {
        const it = _ctx?.tooltip?.dataPoints?.[0];
        if (!it) return colors.value.axisText;
        const i = it.dataIndex;
        const net = (inflowsArr.value[i] ?? 0) - (outflowsArr.value[i] ?? 0);
        return net >= 0 ? colors.value.pos : colors.value.neg;
      },
    },
  },

  scales: {
    x: {
      type: "category",
      grid: { display: false, drawBorder: false },
      ticks: { color: colors.value.axisText, maxRotation: 0, minRotation: 0 },
      border: { color: colors.value.axisBorder },
    },
    y: {
      grid: { display: false, drawBorder: false },
      ticks: {
        display: !props.isMobile,
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
    type="line"
    :data="chartData"
    :options="chartOptions"
    :plugins="[hoverGuidePlugin]"
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

    <span class="text-sm" style="color: var(--text-secondary)">
      Add transactions to see your monthly inflows and outflows.
    </span>
  </div>
</template>
