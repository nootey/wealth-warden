<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useChartStore} from "../../../services/stores/chart_store.ts";
import type {NetworthResponse, ChartPoint} from "../../../models/chart_models.ts";
import SlotSkeleton from "../layout/SlotSkeleton.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import NetworthChart from "../charts/NetworthChart.vue";
import ShowLoading from "../base/ShowLoading.vue";
import {useRouter} from "vue-router";

const props = withDefaults(defineProps<{
    accountId?: number | null
    title?: string
    storageKeyPrefix?: string
    chartHeight: number
}>(), {
    accountId: null,
    title: 'Net worth',
    storageKeyPrefix: 'networth_range_key',
    chartHeight: 300,
})

const router = useRouter();

type RangeKey = '1w'|'1m'|'3m'|'6m'|'ytd'|'1y'|'5y'

const dateRanges = [
    { name: '1W',  key: '1w'  },
    { name: '1M',  key: '1m'  },
    { name: '3M',  key: '3m'  },
    { name: '6M',  key: '6m'  },
    { name: 'YTD', key: 'ytd' },
    { name: '1Y',  key: '1y'  },
    { name: '5Y',  key: '5y'  },
] as const

const periodLabels: Record<RangeKey,string> = {
    '1w': 'week',
    '1m': 'month',
    '3m': '3 months',
    '6m': '6 months',
    'ytd': 'year to date',
    '1y': 'year',
    '5y': '5 years'
}

const toastStore = useToastStore()
const chartStore = useChartStore()

const hydrating = ref(true)
const payload = ref<NetworthResponse | null>(null)
const filteredDateRanges = ref<typeof dateRanges[number][]>([...dateRanges])
const selectedDTO = ref<typeof dateRanges[number] | null>(dateRanges[4]) // default YTD
const selectedKey = computed<RangeKey>(() => (selectedDTO.value?.key ?? 'ytd') as RangeKey)

const orderedPoints = computed<ChartPoint[]>(() => {
    const arr = payload.value?.points ?? []
    return [...arr].sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
})
const hasSeries = computed(() => (payload.value?.points?.length ?? 0) > 0)

const activeColor = ref('#ef4444')

const storageSuffix = computed(() =>
    props.accountId ? `acct_${props.accountId}` : 'ALL'
)
const storageKey = computed(() =>
    `${props.storageKeyPrefix}_${storageSuffix.value}`
)

const startOfRange = computed(() => {
    const today = new Date()
    // normalize to UTC midnight like backend
    const dto = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()))

    switch (selectedKey.value) {
        case '1w': return new Date(dto.getTime() - 7 * 24 * 60 * 60 * 1000)
        case '1m': return new Date(Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 1, dto.getUTCDate()))
        case '3m': return new Date(Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 3, dto.getUTCDate()))
        case '6m': return new Date(Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 6, dto.getUTCDate()))
        case 'ytd': return new Date(Date.UTC(dto.getUTCFullYear(), 0, 1))
        case '1y': return new Date(Date.UTC(dto.getUTCFullYear() - 1, dto.getUTCMonth(), dto.getUTCDate()))
        case '5y': return new Date(Date.UTC(dto.getUTCFullYear() - 5, dto.getUTCMonth(), dto.getUTCDate()))
        default:   return new Date(Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 1, dto.getUTCDate()))
    }
})

const displayPoints = computed<ChartPoint[]>(() => {
    const pts = orderedPoints.value
    if (pts.length === 1) {
        const first = pts[0]
        const start = startOfRange.value
        const firstDay = new Date(first.date)
        // if first point isn't already at start, prepend a phantom point at start with same value
        if (firstDay.getUTCFullYear() !== start.getUTCFullYear() ||
            firstDay.getUTCMonth()  !== start.getUTCMonth()  ||
            firstDay.getUTCDate()   !== start.getUTCDate()) {
            return [
                { date: start.toISOString(), value: first.value },
                ...pts,
            ]
        }
    }
    return pts
})

function searchDaterange(event: any) {
    const q = (event.query ?? '').trim().toLowerCase()
    filteredDateRanges.value = q
        ? dateRanges.filter(o => o.name.toLowerCase().startsWith(q))
        : [...dateRanges]
}

function displayNetworthChange(change: string) {
    if (change === 'year to date') return change
    return `vs. last ${change ?? 'period'}`
}

const pctStr = computed(() => {
    const p = payload.value?.change?.pct ?? 0
    return (p * 100).toFixed(1) + '%'
})

async function getData() {
    const lastKey = localStorage.getItem(storageKey.value)
    if (lastKey) {
        const found = dateRanges.find(r => r.key === lastKey)
        if (found) selectedDTO.value = found
    }
    await getNetworthData({ rangeKey: selectedKey.value })
    hydrating.value = false
}

async function getNetworthData(opts?: { rangeKey?: RangeKey; from?: string; to?: string }) {
    try {
        const params: any = {}
        if (opts?.from || opts?.to) {
            if (opts.from) params.from = opts.from
            if (opts.to) params.to = opts.to
        } else if (opts?.rangeKey) {
            params.range = opts.rangeKey
        }
        if (props.accountId) params.account= props.accountId

        const res = await chartStore.getNetWorth(params)

        res.points = res.points.map((p: any) => ({ ...p, value: Number(p.value) }))
        res.current.value = Number(res.current.value)
        if (res.change) {
            res.change.prev_period_end_value = Number(res.change.prev_period_end_value)
            res.change.current_end_value    = Number(res.change.current_end_value)
            res.change.abs                  = Number(res.change.abs)
            res.change.pct                  = Number(res.change.pct)
        }

        payload.value = res
        activeColor.value = (res?.change?.abs ?? 0) >= 0 ? '#22c55e' : '#ef4444'
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

watch(selectedDTO, (val) => {
    if (!val || hydrating.value) return
    localStorage.setItem(storageKey.value, val.key)
    getNetworthData({ rangeKey: val.key as RangeKey })
})

defineExpose({ refresh: getData })

onMounted(getData)
</script>

<template>
    <SlotSkeleton bg="secondary">
        <div v-if="payload" class="w-full flex flex-column justify-content-center p-3 gap-1">
            <div class="flex flex-row gap-2 w-full justify-content-between">
                <div class="flex flex-column gap-2">
                    <div class="flex flex-row">
                        <span class="text-sm" style="color: var(--text-secondary)">{{ title }}</span>
                    </div>
                    <div class="flex flex-row">
                        <strong>{{ vueHelper.displayAsCurrency(payload.current.value) }}</strong>
                    </div>
                </div>

                <div class="flex flex-column gap-2">
                    <AutoComplete
                            size="small"
                            style="width: 90px;"
                            v-model="selectedDTO"
                            :suggestions="filteredDateRanges"
                            dropdown
                            @complete="searchDaterange"
                            optionLabel="name"
                            forceSelection
                    />
                </div>
            </div>

            <div v-if="payload?.change && hasSeries"
                    class="flex flex-row gap-2 align-items-center"
                    :style="{ color: activeColor }">
                <span>{{ vueHelper.displayAsCurrency(Math.abs(payload.change.abs)) }}</span>

                <div class="flex flex-row gap-1 align-items-center">
                    <i class="text-sm" :class="payload.change.abs >= 0 ? 'pi pi-angle-double-up' : 'pi pi-angle-double-down'"></i>
                    <span>({{ pctStr }})</span>
                </div>

                <span class="text-sm" style="color: var(--text-secondary)">
                    {{ displayNetworthChange(periodLabels[selectedKey]) }}
                </span>
            </div>

            <NetworthChart
                    v-if="hasSeries"
                    :height="chartHeight"
                    :dataPoints="displayPoints"
                    :currency="payload.currency"
                    :activeColor="activeColor"
            />

            <div v-else
                 class="flex flex-column align-items-center justify-content-center border-1 border-dashed border-round-md surface-border"
                 :style="{ height: (chartHeight/2) + 'px' }">
                <i class="pi pi-inbox text-2xl mb-2" style="color: var(--text-secondary)"></i>
                <div class="text-sm" style="color: var(--text-secondary)">
                    <span>
                        No data yet - connect an
                    </span>
                    <span class="hover-icon font-bold text-base" @click="router.push({name: 'accounts'})"> account </span>
                    <span> to see your net worth over time. </span>
                </div>
            </div>

        </div>
        <ShowLoading v-else :numFields="6" />
    </SlotSkeleton>
</template>
