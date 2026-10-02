<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import Chart from "primevue/chart";
import {
  Chart as ChartJS,
  ArcElement,
  Tooltip,
  Legend,
  PieController,
  type ChartOptions,
} from "chart.js";
import {
  categoryPalette,
  useChartColors,
} from "../../../style/theme/chartColors.ts";
import vueHelper from "../../../utils/vue_helper.ts";

ChartJS.register(PieController, ArcElement, Tooltip, Legend);

const props = defineProps<{
  values: number[];
  labels: string[];
  size?: number;
  showLegend?: boolean;
  showTotal?: boolean;
  options?: object;
}>();

const { colors: themeColors } = useChartColors();

const chartRef = ref<any>(null);
const isChartReady = ref(false);

onMounted(() => {
  isChartReady.value = true;
});

onUnmounted(() => {
  chartRef.value?.chart?.destroy?.();
});

const chartData = computed(() => {
  const palette = categoryPalette();
  const colors = Array.from(
    { length: props.labels.length },
    (_, i) => palette[i % palette.length],
  );
  return {
    labels: props.labels,
    datasets: [
      {
        data: props.values,
        backgroundColor: colors,
        borderWidth: 1,
        borderColor: themeColors.value.sliceBorder,
        hoverOffset: 4,
      },
    ],
  };
});

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

const total = computed(() => props.values.reduce((a, b) => a + b, 0));

const baseOptions = computed<ChartOptions<"pie">>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: "62%",
  layout: { padding: 6 },
  animation: {
    animateScale: true,
    animateRotate: true,
    duration: 1500,
    easing: "easeOutCubic",
  },
  plugins: {
    legend: { display: props.showLegend ?? true, position: "top" },
    tooltip: {
      backgroundColor: themeColors.value.ttipBg,
      borderColor: themeColors.value.ttipBorder,
      borderWidth: 1,
      padding: 12,
      cornerRadius: 12,
      boxPadding: 6,
      titleColor: themeColors.value.ttipTitle,
      bodyColor: themeColors.value.ttipText,
      titleFont: { weight: 600, size: 12 },
      bodyFont: { weight: 600, size: 14 },
      callbacks: {
        label: (ctx) => {
          const label = ctx.label ?? "";
          const raw = Number(ctx.parsed);
          const data = (ctx.dataset?.data ?? []) as (number | string)[];
          const total = data.map((v) => Number(v)).reduce((a, b) => a + b, 0);
          const pct = total ? (raw / total) * 100 : 0;
          return `${label}: ${vueHelper.displayAsCurrency(raw)} · ${pct.toFixed(1)} %`;
        },
      },
    },
  },
}));

const chartOptions = computed<ChartOptions<"pie">>(() =>
  merge<ChartOptions<"pie">>(baseOptions.value, props.options ?? {}),
);
</script>

<template>
  <div
    :style="{
      width: (size ?? 300) + 'px',
      height: (size ?? 300) + 'px',
    }"
    class="relative flex justify-center items-center"
  >
    <div
      v-if="showTotal"
      class="absolute inset-0 flex flex-col items-center justify-center gap-1 pointer-events-none"
    >
      <span class="label">Total</span>
      <span class="text-sm font-medium text-ink">
        {{ vueHelper.displayAsCurrency(total) }}
      </span>
    </div>
    <Chart
      v-if="isChartReady"
      ref="chartRef"
      type="pie"
      :data="chartData"
      :options="chartOptions"
      class="relative"
      style="width: 100%; height: 100%"
    />
  </div>
</template>
