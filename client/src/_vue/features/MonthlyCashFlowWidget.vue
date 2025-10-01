<script setup lang="ts">

import MonthlyCashFlowChart from "../components/charts/MonthlyCashFlowChart.vue";
import type {MonthlyCashFlowResponse} from "../../models/chart_models.ts";
import {onMounted, ref, watch} from "vue";
import {useStatisticsStore} from "../../services/stores/statistics_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";

defineProps<{ cashFlow: MonthlyCashFlowResponse }>();

const emit = defineEmits<{
    (e: 'refresh', year: number): void;
}>();

const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const filteredYears = ref<number[]>([]);

const statsStore = useStatisticsStore();
const toastStore = useToastStore();

const loadYears = async () => {
    years.value = await statsStore.getAvailableStatsYears(null);
    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
        ? current
        : (years.value[0] ?? current);
};

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
            emit("refresh", newVal);
        } catch (e) {
            toastStore.errorResponseToast(e);
        }
    }
});

</script>

<template>
    <div class="flex flex-column w-full p-3">
        <div class="flex flex-row gap-2 w-full justify-content-between align-items-center">
            <div class="flex flex-column gap-1">
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

        <MonthlyCashFlowChart v-if="cashFlow.series" :data="cashFlow" />
    </div>
</template>

<style scoped>

</style>