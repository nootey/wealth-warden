<script setup lang="ts">

import MonthlyCashFlowChart from "../components/charts/MonthlyCashFlowChart.vue";
import type {MonthlyCashFlowResponse} from "../../models/chart_models.ts";
import {onMounted, ref, watch} from "vue";
import {useStatisticsStore} from "../../services/stores/statistics_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useChartStore} from "../../services/stores/chart_store.ts";

const statsStore = useStatisticsStore();
const toastStore = useToastStore();
const chartStore = useChartStore();

const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const monthlyCashFlow = ref<MonthlyCashFlowResponse>({ year: 0, series: [] })

onMounted(async () => {
    await fetchMonthlyCashFlows(null);
});

async function fetchMonthlyCashFlows(year: number | null) {

    if(!year) {
        year = new Date().getFullYear();
    }

    try {
        monthlyCashFlow.value = await chartStore.getMonthlyCashFlowForYear({year: year});
    } catch (error) {
        toastStore.errorResponseToast(error)
    }
}

const loadYears = async () => {
    const result = await statsStore.getAvailableStatsYears(null);

    years.value = Array.isArray(result) ? result : [];

    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
        ? current
        : (years.value[0] ?? current);

};

onMounted(async () => {
    try {
        await loadYears();
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
});

watch(selectedYear, async (newVal, oldVal) => {
    if (newVal !== oldVal) {
        try {
            await fetchMonthlyCashFlows(newVal);
        } catch (e) {
            toastStore.errorResponseToast(e);
        }
    }
});

</script>

<template>
    <div class="flex flex-column w-full p-3">

        <div v-if="years.length > 0" id="mobile-row" class="flex flex-row gap-2 w-full justify-content-between align-items-center">

            <div class="flex flex-column gap-1">
                <div class="flex flex-row">
                    <span class="text-sm" style="color: var(--text-secondary)">
                        Select which year you want to display statistics for. Current year will be used as a default.
                    </span>
                </div>
            </div>

            <div class="flex flex-column gap-2">
                <Select size="small"
                        style="width: 100px;"
                        v-model="selectedYear"
                        :options="years"
                />
            </div>
        </div>

        <MonthlyCashFlowChart v-if="monthlyCashFlow.series" :data="monthlyCashFlow" />

    </div>
</template>

<style scoped>

</style>