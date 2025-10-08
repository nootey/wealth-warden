<script setup lang="ts">

import {useStatisticsStore} from "../../../services/stores/statistics_store.ts";
import {onMounted, ref} from "vue";
import type {MonthlyStats} from "../../../models/statistics_models.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";


const statsStore = useStatisticsStore();
const toastStore = useToastStore();

const loading = ref(false);

const monthlyStats = ref<MonthlyStats | null>(null);

onMounted(async () => {
    await loadStats();
})



async function loadStats() {
    try {
        loading.value = true;
        const result = await statsStore.getCurrentMonthsStats(null);

        if (!result) {
            monthlyStats.value = null;
        } else {
            monthlyStats.value = result;
        }

    } catch (e) {
        toastStore.errorResponseToast(e);
    } finally {
        loading.value = false;
    }
}

</script>

<template>
    <div v-if="!loading" class="flex flex-column p-2 gap-2">
        <h4>Accounts</h4>
        <span style="color: var(--text-secondary)">Monthly stats are computed for all checking accounts, which are treated as main accounts.</span>

        <br>

        <div v-if="monthlyStats" class="flex flex-column">
            <div class="flex flex-row gap-2 align-items-center">
                <span>Inflows:</span>
                <span>{{ vueHelper.displayAsCurrency(monthlyStats?.inflow!) }}</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>Outflows:</span>
                <span>{{ vueHelper.displayAsCurrency(monthlyStats?.outflow!) }}</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>Take home:</span>
                <span>{{ vueHelper.displayAsCurrency(monthlyStats?.take_home!) }}</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>Overflow:</span>
                <span>{{ vueHelper.displayAsCurrency(monthlyStats?.overflow!) }}</span>
            </div>

            <br>

            <div class="flex flex-row gap-2 align-items-center">
                <span>Savings:TBD</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>-> Rate: TBD</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>-> Avg. rate: TBD</span>
            </div>

            <div class="flex flex-row gap-2 align-items-center">
                <span>Investments: TBD</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>-> Rate: TBD</span>
            </div>
            <div class="flex flex-row gap-2 align-items-center">
                <span>-> Avg. rate: TBD</span>
            </div>
        </div>
        <div v-else>
            <span>No checking accounts are currently available.</span>
        </div>


    </div>
    <ShowLoading v-else :numFields="7" />
</template>

<style scoped>

</style>