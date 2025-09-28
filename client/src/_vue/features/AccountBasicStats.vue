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

const loadStats = async () => {
    accBasicStats.value = await statsStore.getBasicStatisticsForAccount(
        props.accID ?? null,
        selectedYear.value
    );
};

const loadYears = async () => {
    years.value = await statsStore.getAvailableStatsYears(props.accID ?? null);
    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
        ? current
        : (years.value[0] ?? current);
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
                    const pct = total ? (value / total) * 100 : 0;

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

        <div class="flex flex-row w-full justify-content-center p-1">
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
                    <Carousel v-if="hasInflowData && hasOutflowData"
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
                    <ShowLoading v-else :numFields="4" />
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
    background-color: var(--border-color); /* inactive color */
    border-radius: 50%;
}

:deep([data-p-active="true"] [data-pc-section="indicatorbutton"]) {
    background-color: var(--text-primary); /* active color */
}
</style>