<script setup lang="ts">
import { computed, ref } from 'vue'
import Chart from 'primevue/chart'
import {
    Chart as ChartJS,
    LineController,
    LineElement, PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip, Legend, Filler, CategoryScale
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(
    LineController,
    LineElement, PointElement,
    LinearScale,
    TimeSeriesScale,
    Tooltip, Legend, Filler, CategoryScale
)
type Pt = { date: string; value: number | string }

const props = withDefaults(defineProps<{
    dataPoints: Pt[]
    currency?: string
}>(), {
    dataPoints: () => [],
    currency: 'EUR'
})

const emit = defineEmits<{
    (e: 'point-select', payload: { x: string | number | Date; y: number }): void
}>()

const chartRef = ref<any>(null)
const selected = ref<{ x: string | number | Date; y: number } | null>(null)

const currencyFmt = computed(
    () => new Intl.NumberFormat('de-DE', { style: 'currency', currency: props.currency ?? 'EUR' })
)

const data = computed(() => ({
    datasets: [{
        label: 'Net worth',
        data: props.dataPoints.map(p => ({ date: p.date, value: Number(p.value) })),
        borderWidth: 2,
        pointHoverRadius: 4,
        fill: 'origin',
        tension: 0,
        stepped: 'before',
        spanGaps: true,
        borderColor: 'rgba(99, 102, 241, 1)',
        backgroundColor: 'rgba(99, 102, 241, 0.15)',
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
            callbacks: { label: (ctx: any) => currencyFmt.value.format(ctx.parsed.y) }
        }
    },
    scales: {
        x: {
            type: 'timeseries',
            time: { unit: 'day', tooltipFormat: 'PP' },
            ticks: { maxRotation: 0, autoSkip: true },
            grid: { display: false }
        },
        y: {
            beginAtZero: false,
            ticks: { callback: (v: number | string) => currencyFmt.value.format(Number(v)) }
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
    <div style="height: 320px">
        <Chart ref="chartRef" type="line" :data="data" :options="options" />
    </div>

    <div v-if="selected" style="margin-top: .5rem; font-size: .9rem;">
        Selected:
        <strong>{{ new Date(selected.x).toLocaleDateString() }}</strong>
        — {{ currencyFmt.format(Number(selected.y)) }}
    </div>
</template>
