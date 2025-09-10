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

ChartJS.register(
    LineController,
    LineElement, PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip, Legend, Filler, CategoryScale
)

const props = withDefaults(defineProps<{
    dataPoints: ChartPoint[]
    currency?: string
    activeColor?: string
}>(), {
    dataPoints: () => [],
    currency: 'EUR',
    activeColor: "#ef4444"
})

const emit = defineEmits<{
    (e: 'point-select', payload: { x: string | number | Date; y: number }): void
}>()

const chartRef = ref<any>(null);
const selected = ref<{ x: string | number | Date; y: number } | null>(null);

function hexToRgba(hex: string, alpha = 0.15) {
    // supports #RRGGBB and #RGB
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
        pointHoverRadius: 4,
        fill: false,
        stepped: false,
        tension: 0.35,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderColor: props.activeColor,
        backgroundColor: hexToRgba(props.activeColor, 0.12),
        pointRadius: (ctx: any) => (ctx.dataIndex === selectedIndex.value ? 3 : 0),
    }]
}))

const options = computed(() => ({
    responsive: true,
    maintainAspectRatio: false,
    parsing: { xAxisKey: 'date', yAxisKey: 'value' },
    interaction: { mode: 'nearest', intersect: false },
    plugins: {
        legend: { display: false },
        tooltip: {
            callbacks: { label: (ctx: any) => vueHelper.displayAsCurrency(ctx.parsed.y) }
        }
    },
    scales: {
        x: {
            type: 'timeseries',
            bounds: 'data',
            grid: {
                display: false,
                drawBorder: false
            },
            afterBuildTicks: (scale: any) => {
                const t = scale.ticks
                if (!t?.length) return
                const first = t[0]
                const last  = t[t.length - 1]
                scale.ticks = first.value === last.value ? [first] : [first, last]
            },

            ticks: {
                autoSkip: false,
                maxRotation: 0,
                minRotation: 0,
                color: "grey",
                callback: (_val: any, index: number, ticks: any[]) => {
                    if (index !== 0 && index !== ticks.length - 1) return ''
                    const v = ticks[index].value // ms timestamp
                    return dateHelper.formatDate(v, false, 'MMM D, YYYY', true)
                },
            },

            time: { unit: 'day' }
        },
        y: {
            beginAtZero: false,
            ticks: { display: false },
            grid: {
                display: false,
                drawBorder: false
            }
        }
    },
    onClick: (evt: any, _els: any, chart: any) => {
        const hits = chart.getElementsAtEventForMode(evt, 'nearest', { intersect: false }, true)
        if (hits.length) {
            const { datasetIndex, index } = hits[0]
            const p = chart.data.datasets[datasetIndex].data[index]
            selected.value = { x: p.date, y: p.value }
            emit('point-select', selected.value)
        }
    }
}))

const selectedIndex = computed(() => {
    if (!selected.value) return -1
    const ds = data.value.datasets[0].data as any[]
    return ds.findIndex(p => String(p.date) === String(selected.value!.x))
})
</script>

<template>
    <Chart style="height: 300px;" ref="chartRef" type="line" :data="data" :options="options" />

    <div v-if="selected" style="margin-top: .5rem; font-size: .9rem;">
        Selected:
        <strong>{{ new Date(selected.x).toLocaleDateString() }}</strong>
        — {{ vueHelper.displayAsCurrency(Number(selected.y)) }}
    </div>
</template>
