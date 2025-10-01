<script setup lang="ts">
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import {onMounted, ref} from "vue";
import {useChartStore} from "../../services/stores/chart_store.ts";
import type {MonthlyCashFlowResponse} from "../../models/chart_models.ts";
import MonthlyCashFlowChart from "../components/charts/MonthlyCashFlowChart.vue";

const chartStore = useChartStore();

const monthlyCashFlow = ref<MonthlyCashFlowResponse>({ year: 0, series: [] })

onMounted(async () => {
    monthlyCashFlow.value = await chartStore.getMonthlyCashFlowForYear({year: 2025});
})

</script>

<template>
    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

        <div class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
             style="border: 1px solid var(--border-color); background: var(--background-secondary); max-width: 1000px;">

            <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
                <div style="font-weight: bold;">Charts</div>
            </div>

            <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                <h3>Income vs expense breakdown</h3>
            </div>
            <SlotSkeleton bg="primary">
                <MonthlyCashFlowChart v-if="monthlyCashFlow.series" :data="monthlyCashFlow" />
            </SlotSkeleton>

            <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                <h3>Monthly category display</h3>
            </div>
            <SlotSkeleton bg="primary">
                WIP
            </SlotSkeleton>

        </div>

    </main>
</template>

<style scoped>

</style>