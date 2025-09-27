<script setup lang="ts">
import {computed, onMounted, ref} from "vue";
import type {BasicAccountStats, CategoryStat} from "../../../models/statistics_models.ts";
import {useStatisticsStore} from "../../../services/stores/statistics_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import ShowLoading from "../base/ShowLoading.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import ComparativePieChart from "./ComparativePieChart.vue";

const props = defineProps<{
  accID?: number | null;
  pieChartSize: number;
}>();

const accBasicStats = ref<BasicAccountStats | null>(null);

onMounted(async () => {
    try {
        accBasicStats.value = await statsStore.getBasicStatisticsForAccount(props.accID ?? null, 2025);
    } catch (e) {
        toastStore.errorResponseToast(e);
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

</script>

<template>
    <div v-if="accBasicStats" class="w-full flex flex-column gap-3 p-3">
        <div class="flex flex-row w-full justify-content-center p-1">
            <div class="flex flex-column w-5 gap-3">
                <h4 style="color: var(--text-primary)">Basic</h4>
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

            <div class="flex flex-column w-7 justify-content-center align-items-center gap-2">
                <h4 style="color: var(--text-primary)">Category breakdown</h4>
                <div class="flex flex-column justify-content-center w-10">
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