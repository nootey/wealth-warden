<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  Chart as ChartJS,
  ArcElement,
  Tooltip,
  Legend,
  PieController,
  type ChartOptions,
} from "chart.js";
import { PieChart } from "vue-chart-3";
import type { ChartData } from "chart.js";

// Register necessary Chart.js components
ChartJS.register(PieController, ArcElement, Tooltip, Legend);

const props = defineProps<{
  values: number[];
  labels: string[];
  size?: number;
  showLegend?: boolean;
  options?: object; // generic override
}>();

const chartSize = ref(props.size ?? 300);

const chartData = ref<ChartData<"pie">>({
  labels: props.labels,
  datasets: [
    {
      data: props.values,
      backgroundColor: [
        "#36A2EB",
        "#FF6384",
        "#FFCE56",
        "#4BC0C0",
        "#9966FF",
        "#FF9F40",
      ],
    },
  ],
});

const generateColors = (count: number) => {
  const colors = [
    "#36A2EB",
    "#FF6384",
    "#FFCE56",
    "#4BC0C0",
    "#9966FF",
    "#FF9F40",
  ];
  while (colors.length < count)
    colors.push(`#${Math.floor(Math.random() * 16777215).toString(16)}`);
  return colors.slice(0, count);
};

watch(
  [() => props.values, () => props.labels],
  ([vals, labs]) => {
    chartData.value = {
      labels: labs,
      datasets: [{ data: vals, backgroundColor: generateColors(labs.length) }],
    };
  },
  { immediate: true },
);

function merge<T>(base: any, extra: any): T {
  if (!extra) return base;
  const out = Array.isArray(base) ? [...base] : { ...base };
  for (const k in extra) {
    const bv = (base ?? {})[k];
    const ev = extra[k];
    out[k] =
      bv &&
      typeof bv === "object" &&
      !Array.isArray(bv) &&
      typeof ev === "object" &&
      !Array.isArray(ev)
        ? merge(bv, ev)
        : ev;
  }
  return out as T;
}

// default options
const baseOptions = computed<ChartOptions<"pie">>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: {
    animateScale: true,
    animateRotate: true,
    duration: 1500,
    easing: "easeOutCubic",
  },
  plugins: {
    legend: { display: props.showLegend ?? true, position: "top" },
    tooltip: {
      callbacks: {
        label: (ctx) => {
          const label = ctx.label ?? "";
          const raw = Number(ctx.parsed);
          const data = (ctx.dataset?.data ?? []) as (number | string)[];
          const total = (data as (number | string)[])
            .map((v) => Number(v))
            .reduce((a, b) => a + b, 0);
          const pct = total ? (raw / total) * 100 : 0;
          return `${label}: ${raw.toLocaleString("de-DE", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}€ · ${pct.toFixed(1)} %`;
        },
      },
    },
  },
}));

// final options = base + user overrides (user can replace tooltip label entirely)
const chartOptions = computed<ChartOptions<"pie">>(() =>
  merge<ChartOptions<"pie">>(baseOptions.value, props.options ?? {}),
);
</script>

<template>
  <div
    :style="{
      width: (size ?? chartSize) + 'px',
      height: (size ?? chartSize) + 'px',
    }"
    class="flex justify-content-center align-items-center"
  >
    <PieChart
      :height="chartSize"
      :width="chartSize"
      :chart-data="chartData"
      :options="chartOptions"
    />
  </div>
</template>
