<script setup lang="ts">
import { computed, ref, onUnmounted } from "vue";
import Chart from "primevue/chart";
import {
    Chart as ChartJS,
    BarController, BarElement,
    LinearScale, CategoryScale,
    Tooltip, Legend
} from "chart.js";
import vueHelper from "../../../utils/vue_helper.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";

ChartJS.register(BarController, BarElement, LinearScale, CategoryScale, Tooltip, Legend);

type YearSeries = { name: string; data: number[] };
const props = defineProps<{
    series: YearSeries[];
}>();

const { colors } = useChartColors();
const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

const hoverXByChart = new WeakMap<any, number | null>();
const hoverGuidePlugin = {
    id: "hoverGuide",
    afterEvent(chart: any, args: any) {
        let next: number | null = null;
        if (args.inChartArea) {
            const a = chart.getActiveElements?.() ?? [];
            if (a.length) next = a[0].index ?? null;
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
    }
};

const HEX_PALETTE = [
    '#5B8FF9',
    '#FF9D4D',
    '#36CFC9',
    '#F759AB',
    '#9254DE',
];

const palette = computed(() => HEX_PALETTE);
const chartData = computed(() => ({
    labels: months,
    datasets: props.series.map((s, i) => ({
        label: s.name,
        data: s.data,
        backgroundColor: palette.value[i % palette.value.length],
        borderColor: palette.value[i % palette.value.length],
        borderWidth: 1,
        borderRadius: 6,
        barPercentage: 0.85,
        categoryPercentage: 0.6
    }))
}));

const chartOptions = computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: "index", intersect: false },
    plugins: {
        legend: {
            position: "top",
            labels: {
                color: colors.value.axisText,
                usePointStyle: true,
                pointStyle: "circle",
                boxWidth: 8,
                boxHeight: 8,
                padding: 16
            }
        },
        tooltip: {
            backgroundColor: colors.value.ttipBg,
            borderColor: colors.value.ttipBorder,
            borderWidth: 1,
            padding: 12,
            cornerRadius: 12,
            displayColors: false,
            titleColor: colors.value.ttipTitle,
            bodyColor:  colors.value.ttipText,
            titleFont: { weight: "600", size: 12 },
            bodyFont:  { weight: "600", size: 14 },
            callbacks: {
                title: (items: any[]) => items?.[0]?.label ?? "",
                label: (ctx: any) =>
                    `${ctx.dataset.label}: ${vueHelper.displayAsCurrency(ctx.parsed?.y)}`
            }
        }
    },
    scales: {
        x: {
            type: "category",
            grid: { display: false, drawBorder: false },
            ticks: { color: colors.value.axisText, maxRotation: 0, minRotation: 0 },
            border: { color: colors.value.axisBorder }
        },
        y: {
            grid: { display: false, drawBorder: false },
            ticks: {
                color: colors.value.axisText,
                callback: (v: number) => vueHelper.displayAsCurrency(v)
            },
            border: { color: colors.value.axisBorder }
        }
    }
}));

const chartRef = ref<any>(null);
onUnmounted(() => chartRef.value?.chart?.destroy?.());
</script>

<template>
  <Chart
    ref="chartRef"
    type="bar"
    :data="chartData"
    :options="chartOptions"
    :plugins="[hoverGuidePlugin]"
    style="width: 100%; height: 400px"
  />
</template>