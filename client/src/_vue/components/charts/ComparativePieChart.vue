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
import { CATEGORY_PALETTE } from "../../../style/theme/chartColors.ts";
import vueHelper from "../../../utils/vue_helper.ts";

ChartJS.register(PieController, ArcElement, Tooltip, Legend);

const props = defineProps<{
  values: number[];
  labels: string[];
  size?: number;
  showLegend?: boolean;
  options?: object;
}>();

const chartRef = ref<any>(null);
const isChartReady = ref(false);

onMounted(() => {
  isChartReady.value = true;
});

onUnmounted(() => {
  chartRef.value?.chart?.destroy?.();
});

const chartData = computed(() => {
  const colors = Array.from(
    { length: props.labels.length },
    (_, i) => CATEGORY_PALETTE[i % CATEGORY_PALETTE.length],
  );
  return {
    labels: props.labels,
    datasets: [{ data: props.values, backgroundColor: colors }],
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
    class="flex justify-content-center align-items-center"
  >
    <Chart
      v-if="isChartReady"
      ref="chartRef"
      type="pie"
      :data="chartData"
      :options="chartOptions"
      style="width: 100%; height: 100%"
    />
  </div>
</template>
