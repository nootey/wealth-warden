<script setup lang="ts">
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import {onMounted, ref} from "vue";
import {useChartStore} from "../../services/stores/chart_store.ts";
import type {MonthlyCashFlowResponse} from "../../models/chart_models.ts";
import MonthlyCashFlowWidget from "../features/MonthlyCashFlowWidget.vue";
import {useToastStore} from "../../services/stores/toast_store.ts";

const chartStore = useChartStore();
const toastStore = useToastStore();

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

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case "refreshCashFlow": {
            await fetchMonthlyCashFlows(data);
            break;
        }
        default: break;
    }
}

</script>

<template>
    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

        <div class="flex flex-column justify-content-center p-3 w-full gap-2 border-round-md"
             style="max-width: 1000px;">

            <SlotSkeleton bg="transparent">
                <div class="w-full flex flex-row justify-content-between p-1 gap-2">
                    <div class="w-full flex flex-column gap-2">
                        <h3 style="font-weight: bold;"> Chart view </h3>
                        <div> A better look at your cash-flow. </div>
                    </div>
                </div>
            </SlotSkeleton>

            <div class="w-full flex flex-row justify-content-between p-1">
                <h4>Monthly cash-flow breakdown </h4>
            </div>

            <SlotSkeleton bg="secondary">
                <MonthlyCashFlowWidget :cashFlow="monthlyCashFlow" @refresh="(payload) => handleEmit('refreshCashFlow', payload)" />
            </SlotSkeleton>

            <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                <h4>Monthly category display</h4>
            </div>
            <SlotSkeleton bg="primary">
                WIP
            </SlotSkeleton>
        </div>



    </main>
</template>

<style scoped>

</style>