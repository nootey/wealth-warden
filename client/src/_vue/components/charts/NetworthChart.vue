<script setup lang="ts">
import { computed, markRaw, onMounted, onUnmounted, ref } from "vue";
import Chart from "primevue/chart";

import {
  Chart as ChartJS,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeSeriesScale,
  Tooltip,
  Legend,
  Filler,
  CategoryScale,
} from "chart.js";
import "chartjs-adapter-date-fns";
import dateHelper from "../../../utils/date_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { useThemeStore } from "../../../services/stores/theme_store.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import type {ChartPoint} from "../../../models/analytics_models.ts";

ChartJS.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeSeriesScale,
  Tooltip,
  Legend,
  Filler,
  CategoryScale,
);
const hoverXByChart = new WeakMap<any, number | null>();

const props = withDefaults(
  defineProps<{
    dataPoints?: ChartPoint[];
    currency?: string;
    activeColor?: string;
    height?: number;
    isLiability?: boolean;
  }>(),
  {
    dataPoints: () => [],
    currency: "EUR",
    activeColor: "#ef4444",
    height: 300,
    isLiability: false,
  },
);

defineEmits<{
  (e: "point-select", payload: { x: string | number | Date; y: number }): void;
}>();

const { colors } = useChartColors();

const chartRef = ref<any>(null);

const mql =
  typeof window !== "undefined"
    ? window.matchMedia("(max-width: 640px)")
    : ({
        matches: false,
        addEventListener() {},
        removeEventListener() {},
      } as any);

const pinnedTooltipOnMobile = ref(mql.matches);

const onMqlChange = (e: MediaQueryListEvent) => {
  pinnedTooltipOnMobile.value = e.matches;
};

onMounted(() => {
  const pos = (Tooltip as any).positioners as Record<
    string,
    (items: any[]) => { x: number; y: number }
  >;
  if (!pos.pinned) {
    pos.pinned = function (items: any[]) {
      const chart = items?.[0]?.chart;
      const area = chart?.chartArea;
      if (!chart || !area) return { x: 0, y: 0 };
      const padding = 12;
      const x = area.left + (area.right - area.left) / 2;
      const y = area.top + padding;
      return { x, y };
    };
  }

  mql.addEventListener?.("change", onMqlChange);
});

onUnmounted(() => {
  mql.removeEventListener?.("change", onMqlChange);
  chartRef.value?.chart?.destroy?.();
});

const themeStore = useThemeStore();

const hoverGuidePlugin = markRaw({
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

      // Force scriptable options to be re-evaluated.
      const prevAnim = chart.options.animation;
      chart.options.animation = false;
      chart.draw();
      chart.options.animation = prevAnim;
    }
  },

  afterDatasetsDraw(chart: any, _args: any, opts: any) {
    const hv = hoverXByChart.get(chart);
    if (hv == null) return;

    const { ctx, chartArea, scales } = chart;
    const { top, bottom } = chartArea;
    const x = scales.x.getPixelForValue(hv);

    ctx.save();
    ctx.setLineDash(opts?.dash ?? [4, 6]);
    ctx.lineWidth = opts?.lineWidth ?? 1;
    ctx.strokeStyle = colors.value.guide;
    ctx.beginPath();
    ctx.moveTo(x, top);
    ctx.lineTo(x, bottom);
    ctx.stroke();
    ctx.restore();
  },
});

function hexToRgba(hex: string, alpha = 0.15) {
  const h = hex.replace("#", "");
  const bigint =
    h.length === 3
      ? parseInt(
          h
            .split("")
            .map((c) => c + c)
            .join(""),
          16,
        )
      : parseInt(h, 16);
  const r = (bigint >> 16) & 255;
  const g = (bigint >> 8) & 255;
  const b = bigint & 255;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

const data = computed(() => {
  const d = {
    datasets: [
      {
        label: "Net worth",
        data: props.dataPoints.map((p) => ({
          date: p.date,
          value: Number(p.value),
        })),
        borderWidth: 3,
        pointRadius: 0,
        tension: 0.35,
        cubicInterpolationMode: "monotone",
        spanGaps: true,
        borderColor: props.activeColor,
        backgroundColor: hexToRgba(props.activeColor, 0.12),

        segment: {
          borderColor: (ctx: any) => {
            const hv = hoverXByChart.get(ctx.chart) ?? null;
            if (hv == null) return props.activeColor;
            const x0 = ctx.p0?.parsed?.x;
            return x0 >= hv ? colors.value.dim : props.activeColor;
          },
        },
      },
    ],
  };
  return markRaw(d);
});

const options = computed(() => {
  const o = {
    responsive: true,
    maintainAspectRatio: false,
    parsing: { xAxisKey: "date", yAxisKey: "value" },
    interaction: { mode: "index", intersect: false },
    events: ["mousemove", "mouseout", "touchstart", "touchmove"],
    onClick: undefined,

    plugins: {
      legend: { display: false },
      tooltip: {
        position: pinnedTooltipOnMobile.value ? "pinned" : "average",
        xAlign: "center", // keeps it centered at the pinned x
        yAlign: "top", // keeps it pinned near the top
        caretSize: 0, // cleaner look when pinned
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
          title: (items: any[]) => {
            const v = items?.[0]?.parsed?.x ?? items?.[0]?.raw?.date;
            const ms = typeof v === "number" ? v : new Date(v).getTime();
            return dateHelper.formatDate(ms, false, "MMM D, YYYY", true);
          },

          // current value
          label: (ctx: any) => {
            const y = Number(ctx.parsed?.y ?? ctx.raw?.value);
            const displayY = props.isLiability ? -Math.abs(y) : y;
            return vueHelper.displayAsCurrency(displayY);
          },

          // change vs previous point
          footer: (items: any[]) => {
            const it = items?.[0];
            if (!it) return "";
            const i = it.dataIndex;
            const data = it.dataset?.data || [];
            if (i <= 0 || !data[i - 1]) return "(no previous point)";

            const curr = Number(it.parsed?.y ?? it.raw?.value);
            const prev = Number(data[i - 1]?.value ?? data[i - 1]?.y ?? 0);

            const rawDiff = curr - prev;
            const diff = props.isLiability ? -rawDiff : rawDiff;
            const pct = prev !== 0 ? (diff / Math.abs(prev)) * 100 : null;

            const up = diff >= 0;
            const icon = up ? "+" : "-";

            const absStr = vueHelper.displayAsCurrency(Math.abs(diff));
            const pctStr = pct == null ? "" : ` (${Math.abs(pct).toFixed(1)}%)`;

            return `${icon} ${absStr} ${pctStr}`;
          },
        },

        footerColor: (ctx: any) => {
          const it = ctx?.tooltip?.dataPoints?.[0];
          if (!it) return colors.value.axisText;
          const i = it.dataIndex;
          const data = it.dataset?.data || [];
          if (i <= 0 || !data[i - 1]) return colors.value.axisText;

          const curr = Number(it.parsed?.y ?? it.raw?.value);
          const prev = Number(data[i - 1]?.value ?? data[i - 1]?.y ?? 0);
          const rawDiff = curr - prev;
          const diff = props.isLiability ? -rawDiff : rawDiff;
          return diff >= 0 ? colors.value.pos : colors.value.neg;
        },
      },
      hoverGuide: {
        dashColor: "rgba(255,255,255,0.35)",
        dash: [4, 6],
        lineWidth: 1,
      },
    },

    scales: {
      x: {
        type: "timeseries",
        bounds: "data",
        grid: { display: false, drawBorder: false },
        afterBuildTicks: (scale: any) => {
          const t = scale.ticks;
          if (!t?.length) return;
          const first = t[0],
            last = t[t.length - 1];
          scale.ticks = first.value === last.value ? [first] : [first, last];
        },
        ticks: {
          autoSkip: false,
          maxRotation: 0,
          minRotation: 0,
          color: colors.value.axisText,
          callback: (_: any, i: number, ticks: any[]) =>
            i !== 0 && i !== ticks.length - 1
              ? ""
              : dateHelper.formatDate(
                  ticks[i].value,
                  false,
                  "MMM D, YYYY",
                  true,
                ),
        },
        time: { unit: "day" },
        border: { color: colors.value.axisBorder },
      },
      y: {
        beginAtZero: false,
        ticks: { display: false },
        grid: { display: false, drawBorder: false },
        border: { color: colors.value.axisBorder },
      },
    },
  };
  return markRaw(o);
});
</script>

<template>
  <Chart
    ref="chartRef"
    :key="`nw-${themeStore.isDark}-${dataPoints?.length}-${pinnedTooltipOnMobile}`"
    type="line"
    :data="data"
    :options="options"
    :plugins="[hoverGuidePlugin]"
    :style="{ height: height + 'px' }"
  />
</template>
