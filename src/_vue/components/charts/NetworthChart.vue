<script setup lang="ts">
import {computed, ref} from 'vue'
import Chart from 'primevue/chart'
import type {ChartPoint} from "../../../models/chart_models.ts";

import {
    Chart as ChartJS,
    LineController,
    LineElement, PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip, Legend, Filler, CategoryScale
} from 'chart.js'
import 'chartjs-adapter-date-fns'
import dateHelper from "../../../utils/date_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import {useThemeStore} from "../../../services/stores/theme_store.ts";

ChartJS.register(
    LineController,
    LineElement, PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip, Legend, Filler, CategoryScale
)

const themeStore = useThemeStore();

const dimColor = themeStore.darkModeActive ? hexToRgba("#9C9C9C") : hexToRgba("#1C1919")

const hoverXByChart = new WeakMap<any, number | null>()

const hoverGuidePlugin = {
    id: 'hoverGuide',

    afterEvent(chart: any, args: any) {
        let next: number | null = null
        if (args.inChartArea) {
            const a = chart.getActiveElements?.() ?? []
            if (a.length) next = a[0].element?.$context?.parsed?.x ?? null
        }
        if (args.event?.type === 'mouseout') next = null

        const prev = hoverXByChart.get(chart) ?? null
        if (prev !== next) {
            hoverXByChart.set(chart, next)
            chart.update('none')
        }
    },

    afterDatasetsDraw(chart: any, _args: any, opts: any) {
        const hv = hoverXByChart.get(chart)
        if (hv == null) return

        const { ctx, chartArea, scales } = chart
        const { top, bottom } = chartArea
        const x = scales.x.getPixelForValue(hv)

        ctx.save()
        ctx.setLineDash(opts?.dash ?? [4, 6])
        ctx.lineWidth = opts?.lineWidth ?? 1
        ctx.strokeStyle = opts?.dashColor ?? dimColor
        ctx.beginPath()
        ctx.moveTo(x, top)
        ctx.lineTo(x, bottom)
        ctx.stroke()
        ctx.restore()
    }
}

ChartJS.register(hoverGuidePlugin)

const props = withDefaults(defineProps<{
    dataPoints: ChartPoint[]
    currency?: string
    activeColor?: string
    height: number
}>(), {
    dataPoints: () => [],
    currency: 'EUR',
    activeColor: "#ef4444",
    height: 300,
})

defineEmits<{
    (e: 'point-select', payload: { x: string | number | Date; y: number }): void
}>()

const chartRef = ref<any>(null);
const selected = ref<{ x: string | number | Date; y: number } | null>(null);

function hexToRgba(hex: string, alpha = 0.15) {
    const h = hex.replace('#', '')
    const bigint = h.length === 3
        ? parseInt(h.split('').map(c => c + c).join(''), 16)
        : parseInt(h, 16)
    const r = (bigint >> 16) & 255
    const g = (bigint >> 8) & 255
    const b = bigint & 255
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

const data = computed(() => ({
    datasets: [{
        label: 'Net worth',
        data: props.dataPoints.map(p => ({ date: p.date, value: Number(p.value) })),
        borderWidth: 3,
        pointRadius: 0,
        tension: 0.35,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderColor: props.activeColor,
        backgroundColor: hexToRgba(props.activeColor, 0.12),

        segment: {
            borderColor: (ctx: any) => {
                const hv = hoverXByChart.get(ctx.chart) ?? null
                if (hv == null) return props.activeColor
                const x0 = ctx.p0?.parsed?.x
                return x0 >= hv ? dimColor : props.activeColor
            }
        }
    }]
}))

const options = computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    parsing: { xAxisKey: 'date', yAxisKey: 'value' },
    interaction: { mode: 'nearest', intersect: false },
    events: ['mousemove', 'mouseout', 'touchstart', 'touchmove'],
    onClick: undefined,

    plugins: {
        legend: { display: false },
        tooltip: {
            displayColors: false,
            callbacks: {
                title: (items: any[]) => {
                    const v = items?.[0]?.parsed?.x ?? items?.[0]?.raw?.date
                    const ms = typeof v === 'number' ? v : new Date(v).getTime()
                    return dateHelper.formatDate(ms, false, 'MMM D, YYYY', true)
                },
                label: (ctx: any) => vueHelper.displayAsCurrency(ctx.parsed.y),
            }
        },
        hoverGuide: {
            dashColor: 'rgba(255,255,255,0.35)',
            dash: [4, 6],
            lineWidth: 1
        }
    },

    scales: {
        x: {
            type: 'timeseries',
            bounds: 'data',
            grid: { display: false, drawBorder: false },
            afterBuildTicks: (scale: any) => {
                const t = scale.ticks; if (!t?.length) return
                const first = t[0], last = t[t.length - 1]
                scale.ticks = first.value === last.value ? [first] : [first, last]
            },
            ticks: {
                autoSkip: false,
                maxRotation: 0,
                minRotation: 0,
                color: 'grey',
                callback: (_: any, i: number, ticks: any[]) =>
                    (i !== 0 && i !== ticks.length - 1) ? '' :
                        dateHelper.formatDate(ticks[i].value, false, 'MMM D, YYYY', true),
            },
            time: { unit: 'day' }
        },
        y: {
            beginAtZero: false,
            ticks: { display: false },
            grid: { display: false, drawBorder: false }
        }
    }
}))

</script>

<template>
    <Chart :style="{ height: height + 'px' }" ref="chartRef" type="line" :data="data" :options="options" />

    <div v-if="selected" style="margin-top: .5rem; font-size: .9rem;">
        Selected:
        <strong>{{ new Date(selected.x).toLocaleDateString() }}</strong>
        — {{ vueHelper.displayAsCurrency(Number(selected.y)) }}
    </div>
</template>
