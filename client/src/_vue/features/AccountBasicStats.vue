<script setup lang="ts">
import {computed, onMounted, ref, watch} from "vue";
import type {BasicAccountStats, CategoryStat} from "../../models/statistics_models.ts";
import {useStatisticsStore} from "../../services/stores/statistics_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import ComparativePieChart from "../components/charts/ComparativePieChart.vue";

const props = defineProps<{
  accID?: number | null;
  pieChartSize: number;
}>();

const accBasicStats = ref<BasicAccountStats | null>(null);
const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const filteredYears = ref<number[]>([]);

const isLoadingYears = ref(false);
const isLoadingStats = ref(false);
const isLoading = computed(() => isLoadingYears.value || isLoadingStats.value);

const loadStats = async () => {
    isLoadingStats.value = true;
    try {
        const res = await statsStore.getBasicStatisticsForAccount(
            props.accID ?? null,
            selectedYear.value
        );

        accBasicStats.value = res;
    } finally {
        isLoadingStats.value = false;
    }
};

const loadYears = async () => {
    isLoadingYears.value = true;
    try {
        const result = await statsStore.getAvailableStatsYears(props.accID ?? null);
        years.value = Array.isArray(result) ? result : [];

        const current = new Date().getFullYear();
        selectedYear.value = years.value.includes(current)
            ? current
            : (years.value[0] ?? current);

        filteredYears.value = [...years.value];
    } finally {
        isLoadingYears.value = false;
    }
};

onMounted(async () => {
    try {
        await loadYears();
        await loadStats();
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
});

watch(selectedYear, async (newVal, oldVal) => {
    if (newVal !== oldVal) {
        try {
            await loadStats();
        } catch (e) {
            toastStore.errorResponseToast(e);
        }
    }
});

const statsStore = useStatisticsStore();
const toastStore = useToastStore();

const toNumber = (val?: string | null) => {
    if (val == null) return 0;
    const n = Number(val);
    return isNaN(n) ? 0 : n;
};

const inflowLabels = computed<string[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.inflow) > 0)
        .map((c: CategoryStat) => c.category_name ?? `Category ${c.category_id}`);
});

const inflowValues = computed<number[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.inflow) > 0)
        .map((c: CategoryStat) => toNumber(c.inflow));
});

const outflowLabels = computed<string[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.outflow) !== 0)
        .map((c: CategoryStat) => c.category_name ?? `Category ${c.category_id}`);
});

const outflowValues = computed<number[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.outflow) !== 0)
        .map((c: CategoryStat) => Math.abs(toNumber(c.outflow)));
});

const chartItems = [
    { type: "inflows" },
    { type: "outflows" },
];

const hasInflowData = computed(() => inflowValues.value?.length > 0);
const hasOutflowData = computed(() => outflowValues.value?.length > 0);

const pieOptions = computed(() => ({
    plugins: {
        legend: { display: false, position: "bottom" },
        tooltip: {
            callbacks: {
                label: (ctx: any) => {
                    const label = ctx.label ?? "";
                    const value = Number(ctx.parsed);
                    const data = (ctx.dataset?.data ?? []) as (number | string)[];
                    const total = (data as (number | string)[])
                        .map(v => Number(v))
                        .reduce((a, b) => a + b, 0);
                    const pct = total ? (value / total) : 0;

                    return `${label}: ${vueHelper.displayAsCurrency(value)} · ${vueHelper.displayAsPercentage(pct)}`;
                }
            }
        }
    }
}));

function searchYear(event: any) {
    const q = String((event?.query ?? "")).trim().toLowerCase();
    if (!q) {
        filteredYears.value = [...years.value];
        return;
    }
    filteredYears.value = years.value.filter((y) =>
        String(y).toLowerCase().includes(q)
    );
}

</script>

<template>
    <div v-if="accBasicStats" class="w-full flex flex-column gap-2 p-3">

        <h3 style="color: var(--text-primary)">Basic</h3>
        <div class="flex flex-row gap-2 w-full justify-content-between align-items-center">
            <div class="flex flex-column gap-2">
                <div class="flex flex-row">
                    <span class="text-sm" style="color: var(--text-secondary)">
                        Select which year you want to display statistics for. Current year will be used as a default.
                    </span>
                </div>
            </div>

            <div class="flex flex-column gap-2">
                <AutoComplete
                        size="small"
                        style="width: 100px;"
                        v-model="selectedYear"
                        :suggestions="filteredYears"
                        dropdown
                        @complete="searchYear"
                        forceSelection
                />
            </div>
        </div>

        <div id="stats-row" class="flex flex-row w-full justify-content-center p-1">
            <div class="flex flex-column w-6 gap-3">
                <div class="flex flex-row gap-2">
                    <span>Total inflows:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.inflow) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Total outflows:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.outflow) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Avg. monthly inflows:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_inflow) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Avg. monthly outflows:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_outflow) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Take home:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.take_home) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Overflow:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.overflow) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Avg. monthly take home:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_take_home) }}</b>
                </div>
                <div class="flex flex-row gap-2">
                    <span>Avg. monthly overflow:</span>
                    <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_overflow) }}</b>
                </div>
            </div>
            <div class="flex flex-column w-6 justify-content-center align-items-center gap-2">
                <div class="flex flex-column justify-content-center w-12">
                    <ShowLoading v-if="isLoading" :numFields="4" />
                    <template v-else>
                        <Carousel v-if="hasInflowData && hasOutflowData" id="stats-carousel"
                                  :value="chartItems" :numVisible="1" :numScroll="1">
                            <template #item="slotProps">
                                <div class="flex flex-column justify-content-center align-items-center">
                                    {{ vueHelper.capitalize(slotProps.data.type) }}
                                </div>
                                <div class="flex flex-column justify-content-center align-items-center">
                                    <ComparativePieChart
                                            v-if="slotProps.data.type === 'inflows'"
                                            :size="pieChartSize"
                                            :showLegend="false"
                                            :options="pieOptions"
                                            :values="inflowValues"
                                            :labels="inflowLabels"
                                    />
                                    <ComparativePieChart
                                            v-else
                                            :size="pieChartSize"
                                            :showLegend="false"
                                            :options="pieOptions"
                                            :values="outflowValues"
                                            :labels="outflowLabels"
                                    />
                                </div>
                            </template>
                        </Carousel>
                        <div v-else class="flex flex-column align-items-center justify-content-center p-3"
                             style="border: 1px dashed var(--border-color); border-radius: 16px;">
                              <span class="text-sm" style="color: var(--text-secondary);">
                                    Not enough transactions found for {{ selectedYear }}.
                              </span>
                             <span class="text-sm" style="color: var(--text-secondary);">
                                    Keep inserting transactions to see the chart.
                              </span>
                        </div>
                    </template>

                </div>
            </div>
        </div>

    </div>
    <ShowLoading v-else :numFields="5" />
</template>

<style scoped>

:deep([data-pc-section="indicatorlist"]) { margin-top: 6px; gap: 6px; }
:deep([data-pc-section="content"]) { padding: 0; }
:deep([data-pc-section="indicatorbutton"]) {
    transform: scale(0.6);
    background-color: var(--border-color);
    border-radius: 50%;
}
:deep([data-p-active="true"] [data-pc-section="indicatorbutton"]) {
    background-color: var(--text-primary);
}

@media (max-width: 768px) {

    #stats-row {
        flex-direction: column !important;
        align-items: stretch !important;
        min-width: 0 !important;
    }
    #stats-row > div {
        width: 100% !important;
        min-width: 0 !important;
    }
    #stats-row > div:last-child { margin-top: 1rem; }

    :deep(#stats-carousel button[aria-label="Previous Page"]),
    :deep(#stats-carousel button[aria-label="Next Page"]),
    :deep(#stats-carousel button[data-pc-group="navigator"]) {
        display: none !important;
    }

    :deep(#stats-carousel .flex.justify-content-center.align-items-center) {
        width: 100% !important;
        height: auto !important;
        padding: 0 !important;
        transform: scale(0.9);
        transform-origin: center top;
    }

    :deep(#stats-carousel canvas) {
        width: 100% !important;
        height: auto !important;
    }
}
</style>